/*--------------------------------------------------------------------
 * Symbols referenced in this file:
 * - NameListToString
 * - get_collation_oid
 *--------------------------------------------------------------------
 */

/*-------------------------------------------------------------------------
 *
 * namespace.c
 *	  code to support accessing and searching namespaces
 *
 * This is separate from pg_namespace.c, which contains the routines that
 * directly manipulate the pg_namespace system catalog.  This module
 * provides routines associated with defining a "namespace search path"
 * and implementing search-path-controlled searches.
 *
 *
 * Portions Copyright (c) 1996-2024, PostgreSQL Global Development Group
 * Portions Copyright (c) 1994, Regents of the University of California
 *
 * IDENTIFICATION
 *	  src/backend/catalog/namespace.c
 *
 *-------------------------------------------------------------------------
 */
#include "postgres.h"

#include "access/htup_details.h"
#include "access/parallel.h"
#include "access/xact.h"
#include "access/xlog.h"
#include "catalog/dependency.h"
#include "catalog/namespace.h"
#include "catalog/objectaccess.h"
#include "catalog/pg_authid.h"
#include "catalog/pg_collation.h"
#include "catalog/pg_conversion.h"
#include "catalog/pg_database.h"
#include "catalog/pg_namespace.h"
#include "catalog/pg_opclass.h"
#include "catalog/pg_operator.h"
#include "catalog/pg_opfamily.h"
#include "catalog/pg_proc.h"
#include "catalog/pg_statistic_ext.h"
#include "catalog/pg_ts_config.h"
#include "catalog/pg_ts_dict.h"
#include "catalog/pg_ts_parser.h"
#include "catalog/pg_ts_template.h"
#include "catalog/pg_type.h"
#include "commands/dbcommands.h"
#include "common/hashfn_unstable.h"
#include "funcapi.h"
#include "mb/pg_wchar.h"
#include "miscadmin.h"
#include "nodes/makefuncs.h"
#include "storage/ipc.h"
#include "storage/lmgr.h"
#include "storage/procarray.h"
#include "utils/acl.h"
#include "utils/builtins.h"
#include "utils/catcache.h"
#include "utils/guc_hooks.h"
#include "utils/inval.h"
#include "utils/lsyscache.h"
#include "utils/memutils.h"
#include "utils/snapmgr.h"
#include "utils/syscache.h"
#include "utils/varlena.h"


/*
 * The namespace search path is a possibly-empty list of namespace OIDs.
 * In addition to the explicit list, implicitly-searched namespaces
 * may be included:
 *
 * 1. If a TEMP table namespace has been initialized in this session, it
 * is implicitly searched first.
 *
 * 2. The system catalog namespace is always searched.  If the system
 * namespace is present in the explicit path then it will be searched in
 * the specified order; otherwise it will be searched after TEMP tables and
 * *before* the explicit list.  (It might seem that the system namespace
 * should be implicitly last, but this behavior appears to be required by
 * SQL99.  Also, this provides a way to search the system namespace first
 * without thereby making it the default creation target namespace.)
 *
 * For security reasons, searches using the search path will ignore the temp
 * namespace when searching for any object type other than relations and
 * types.  (We must allow types since temp tables have rowtypes.)
 *
 * The default creation target namespace is always the first element of the
 * explicit list.  If the explicit list is empty, there is no default target.
 *
 * The textual specification of search_path can include "$user" to refer to
 * the namespace named the same as the current user, if any.  (This is just
 * ignored if there is no such namespace.)	Also, it can include "pg_temp"
 * to refer to the current backend's temp namespace.  This is usually also
 * ignorable if the temp namespace hasn't been set up, but there's a special
 * case: if "pg_temp" appears first then it should be the default creation
 * target.  We kluge this case a little bit so that the temp namespace isn't
 * set up until the first attempt to create something in it.  (The reason for
 * klugery is that we can't create the temp namespace outside a transaction,
 * but initial GUC processing of search_path happens outside a transaction.)
 * activeTempCreationPending is true if "pg_temp" appears first in the string
 * but is not reflected in activeCreationNamespace because the namespace isn't
 * set up yet.
 *
 * In bootstrap mode, the search path is set equal to "pg_catalog", so that
 * the system namespace is the only one searched or inserted into.
 * initdb is also careful to set search_path to "pg_catalog" for its
 * post-bootstrap standalone backend runs.  Otherwise the default search
 * path is determined by GUC.  The factory default path contains the PUBLIC
 * namespace (if it exists), preceded by the user's personal namespace
 * (if one exists).
 *
 * activeSearchPath is always the actually active path; it points to
 * baseSearchPath which is the list derived from namespace_search_path.
 *
 * If baseSearchPathValid is false, then baseSearchPath (and other derived
 * variables) need to be recomputed from namespace_search_path, or retrieved
 * from the search path cache if there haven't been any syscache
 * invalidations.  We mark it invalid upon an assignment to
 * namespace_search_path or receipt of a syscache invalidation event for
 * pg_namespace or pg_authid.  The recomputation is done during the next
 * lookup attempt.
 *
 * Any namespaces mentioned in namespace_search_path that are not readable
 * by the current user ID are simply left out of baseSearchPath; so
 * we have to be willing to recompute the path when current userid changes.
 * namespaceUser is the userid the path has been computed for.
 *
 * Note: all data pointed to by these List variables is in TopMemoryContext.
 *
 * activePathGeneration is incremented whenever the effective values of
 * activeSearchPath/activeCreationNamespace/activeTempCreationPending change.
 * This can be used to quickly detect whether any change has happened since
 * a previous examination of the search path state.
 */

/* These variables define the actually active state: */



/* default place to create stuff; if InvalidOid, no default */


/* if true, activeCreationNamespace is wrong, it should be temp namespace */


/* current generation counter; make sure this is never zero */


/* These variables are the values last derived from namespace_search_path: */









/* The above four values are valid only if baseSearchPathValid */


/*
 * Storage for search path cache.  Clear searchPathCacheValid as a simple
 * way to invalidate *all* the cache entries, not just the active one.
 */



typedef struct SearchPathCacheKey
{
	const char *searchPath;
	Oid			roleid;
} SearchPathCacheKey;

typedef struct SearchPathCacheEntry
{
	SearchPathCacheKey key;
	List	   *oidlist;		/* namespace OIDs that pass ACL checks */
	List	   *finalPath;		/* cached final computed search path */
	Oid			firstNS;		/* first explicitly-listed namespace */
	bool		temp_missing;
	bool		forceRecompute; /* force recompute of finalPath */

	/* needed for simplehash */
	char		status;
} SearchPathCacheEntry;

/*
 * myTempNamespace is InvalidOid until and unless a TEMP namespace is set up
 * in a particular backend session (this happens when a CREATE TEMP TABLE
 * command is first executed).  Thereafter it's the OID of the temp namespace.
 *
 * myTempToastNamespace is the OID of the namespace for my temp tables' toast
 * tables.  It is set when myTempNamespace is, and is InvalidOid before that.
 *
 * myTempNamespaceSubID shows whether we've created the TEMP namespace in the
 * current subtransaction.  The flag propagates up the subtransaction tree,
 * so the main transaction will correctly recognize the flag if all
 * intermediate subtransactions commit.  When it is InvalidSubTransactionId,
 * we either haven't made the TEMP namespace yet, or have successfully
 * committed its creation, depending on whether myTempNamespace is valid.
 */






/*
 * This is the user's textual search path specification --- it's the value
 * of the GUC variable 'search_path'.
 */



/* Local functions */
static bool RelationIsVisibleExt(Oid relid, bool *is_missing);
static bool TypeIsVisibleExt(Oid typid, bool *is_missing);
static bool FunctionIsVisibleExt(Oid funcid, bool *is_missing);
static bool OperatorIsVisibleExt(Oid oprid, bool *is_missing);
static bool OpclassIsVisibleExt(Oid opcid, bool *is_missing);
static bool OpfamilyIsVisibleExt(Oid opfid, bool *is_missing);
static bool CollationIsVisibleExt(Oid collid, bool *is_missing);
static bool ConversionIsVisibleExt(Oid conid, bool *is_missing);
static bool StatisticsObjIsVisibleExt(Oid stxid, bool *is_missing);
static bool TSParserIsVisibleExt(Oid prsId, bool *is_missing);
static bool TSDictionaryIsVisibleExt(Oid dictId, bool *is_missing);
static bool TSTemplateIsVisibleExt(Oid tmplId, bool *is_missing);
static bool TSConfigIsVisibleExt(Oid cfgid, bool *is_missing);
static void recomputeNamespacePath(void);
static void AccessTempTableNamespace(bool force);
static void InitTempTableNamespace(void);
static void RemoveTempRelations(Oid tempNamespaceId);
static void RemoveTempRelationsCallback(int code, Datum arg);
static void InvalidationCallback(Datum arg, int cacheid, uint32 hashvalue);
static bool MatchNamedCall(HeapTuple proctup, int nargs, List *argnames,
						   bool include_out_arguments, int pronargs,
						   int **argnumbers);

/*
 * Recomputing the namespace path can be costly when done frequently, such as
 * when a function has search_path set in proconfig. Add a search path cache
 * that can be used by recomputeNamespacePath().
 *
 * The cache is also used to remember already-validated strings in
 * check_search_path() to avoid the need to call SplitIdentifierString()
 * repeatedly.
 *
 * The search path cache is based on a wrapper around a simplehash hash table
 * (nsphash, defined below). The spcache wrapper deals with OOM while trying
 * to initialize a key, optimizes repeated lookups of the same key, and also
 * offers a more convenient API.
 */





#define SH_PREFIX		nsphash
#define SH_ELEMENT_TYPE	SearchPathCacheEntry
#define SH_KEY_TYPE		SearchPathCacheKey
#define SH_KEY			key
#define SH_HASH_KEY(tb, key)   	spcachekey_hash(key)
#define SH_EQUAL(tb, a, b)		spcachekey_equal(a, b)
#define SH_SCOPE		static inline
#define SH_DECLARE
// #define SH_DEFINE
#include "lib/simplehash.h"

/*
 * We only expect a small number of unique search_path strings to be used. If
 * this cache grows to an unreasonable size, reset it to avoid steady-state
 * memory growth. Most likely, only a few of those entries will benefit from
 * the cache, and the cache will be quickly repopulated with such entries.
 */
#define SPCACHE_RESET_THRESHOLD		256




/*
 * Create or reset search_path cache as necessary.
 */


/*
 * Look up entry in search path cache without inserting. Returns NULL if not
 * present.
 */


/*
 * Look up or insert entry in search path cache.
 *
 * Initialize key safely, so that OOM does not leave an entry without a valid
 * key. Caller must ensure that non-key contents are properly initialized.
 */


/*
 * RangeVarGetRelidExtended
 *		Given a RangeVar describing an existing relation,
 *		select the proper namespace and look up the relation OID.
 *
 * If the schema or relation is not found, return InvalidOid if flags contains
 * RVR_MISSING_OK, otherwise raise an error.
 *
 * If flags contains RVR_NOWAIT, throw an error if we'd have to wait for a
 * lock.
 *
 * If flags contains RVR_SKIP_LOCKED, return InvalidOid if we'd have to wait
 * for a lock.
 *
 * flags cannot contain both RVR_NOWAIT and RVR_SKIP_LOCKED.
 *
 * Note that if RVR_MISSING_OK and RVR_SKIP_LOCKED are both specified, a
 * return value of InvalidOid could either mean the relation is missing or it
 * could not be locked.
 *
 * Callback allows caller to check permissions or acquire additional locks
 * prior to grabbing the relation lock.
 */


/*
 * RangeVarGetCreationNamespace
 *		Given a RangeVar describing a to-be-created relation,
 *		choose which namespace to create it in.
 *
 * Note: calling this may result in a CommandCounterIncrement operation.
 * That will happen on the first request for a temp table in any particular
 * backend run; we will need to either create or clean out the temp schema.
 */


/*
 * RangeVarGetAndCheckCreationNamespace
 *
 * This function returns the OID of the namespace in which a new relation
 * with a given name should be created.  If the user does not have CREATE
 * permission on the target namespace, this function will instead signal
 * an ERROR.
 *
 * If non-NULL, *existing_relation_id is set to the OID of any existing relation
 * with the same name which already exists in that namespace, or to InvalidOid
 * if no such relation exists.
 *
 * If lockmode != NoLock, the specified lock mode is acquired on the existing
 * relation, if any, provided that the current user owns the target relation.
 * However, if lockmode != NoLock and the user does not own the target
 * relation, we throw an ERROR, as we must not try to lock relations the
 * user does not have permissions on.
 *
 * As a side effect, this function acquires AccessShareLock on the target
 * namespace.  Without this, the namespace could be dropped before our
 * transaction commits, leaving behind relations with relnamespace pointing
 * to a no-longer-existent namespace.
 *
 * As a further side-effect, if the selected namespace is a temporary namespace,
 * we mark the RangeVar as RELPERSISTENCE_TEMP.
 */


/*
 * Adjust the relpersistence for an about-to-be-created relation based on the
 * creation namespace, and throw an error for invalid combinations.
 */


/*
 * RelnameGetRelid
 *		Try to resolve an unqualified relation name.
 *		Returns OID if relation found in search path, else InvalidOid.
 */



/*
 * RelationIsVisible
 *		Determine whether a relation (identified by OID) is visible in the
 *		current search path.  Visible means "would be found by searching
 *		for the unqualified relation name".
 */


/*
 * RelationIsVisibleExt
 *		As above, but if the relation isn't found and is_missing is not NULL,
 *		then set *is_missing = true and return false instead of throwing
 *		an error.  (Caller must initialize *is_missing = false.)
 */



/*
 * TypenameGetTypid
 *		Wrapper for binary compatibility.
 */


/*
 * TypenameGetTypidExtended
 *		Try to resolve an unqualified datatype name.
 *		Returns OID if type found in search path, else InvalidOid.
 *
 * This is essentially the same as RelnameGetRelid.
 */


/*
 * TypeIsVisible
 *		Determine whether a type (identified by OID) is visible in the
 *		current search path.  Visible means "would be found by searching
 *		for the unqualified type name".
 */


/*
 * TypeIsVisibleExt
 *		As above, but if the type isn't found and is_missing is not NULL,
 *		then set *is_missing = true and return false instead of throwing
 *		an error.  (Caller must initialize *is_missing = false.)
 */



/*
 * FuncnameGetCandidates
 *		Given a possibly-qualified function name and argument count,
 *		retrieve a list of the possible matches.
 *
 * If nargs is -1, we return all functions matching the given name,
 * regardless of argument count.  (argnames must be NIL, and expand_variadic
 * and expand_defaults must be false, in this case.)
 *
 * If argnames isn't NIL, we are considering a named- or mixed-notation call,
 * and only functions having all the listed argument names will be returned.
 * (We assume that length(argnames) <= nargs and all the passed-in names are
 * distinct.)  The returned structs will include an argnumbers array showing
 * the actual argument index for each logical argument position.
 *
 * If expand_variadic is true, then variadic functions having the same number
 * or fewer arguments will be retrieved, with the variadic argument and any
 * additional argument positions filled with the variadic element type.
 * nvargs in the returned struct is set to the number of such arguments.
 * If expand_variadic is false, variadic arguments are not treated specially,
 * and the returned nvargs will always be zero.
 *
 * If expand_defaults is true, functions that could match after insertion of
 * default argument values will also be retrieved.  In this case the returned
 * structs could have nargs > passed-in nargs, and ndargs is set to the number
 * of additional args (which can be retrieved from the function's
 * proargdefaults entry).
 *
 * If include_out_arguments is true, then OUT-mode arguments are considered to
 * be included in the argument list.  Their types are included in the returned
 * arrays, and argnumbers are indexes in proallargtypes not proargtypes.
 * We also set nominalnargs to be the length of proallargtypes not proargtypes.
 * Otherwise OUT-mode arguments are ignored.
 *
 * It is not possible for nvargs and ndargs to both be nonzero in the same
 * list entry, since default insertion allows matches to functions with more
 * than nargs arguments while the variadic transformation requires the same
 * number or less.
 *
 * When argnames isn't NIL, the returned args[] type arrays are not ordered
 * according to the functions' declarations, but rather according to the call:
 * first any positional arguments, then the named arguments, then defaulted
 * arguments (if needed and allowed by expand_defaults).  The argnumbers[]
 * array can be used to map this back to the catalog information.
 * argnumbers[k] is set to the proargtypes or proallargtypes index of the
 * k'th call argument.
 *
 * We search a single namespace if the function name is qualified, else
 * all namespaces in the search path.  In the multiple-namespace case,
 * we arrange for entries in earlier namespaces to mask identical entries in
 * later namespaces.
 *
 * When expanding variadics, we arrange for non-variadic functions to mask
 * variadic ones if the expanded argument list is the same.  It is still
 * possible for there to be conflicts between different variadic functions,
 * however.
 *
 * It is guaranteed that the return list will never contain multiple entries
 * with identical argument lists.  When expand_defaults is true, the entries
 * could have more than nargs positions, but we still guarantee that they are
 * distinct in the first nargs positions.  However, if argnames isn't NIL or
 * either expand_variadic or expand_defaults is true, there might be multiple
 * candidate functions that expand to identical argument lists.  Rather than
 * throw error here, we report such situations by returning a single entry
 * with oid = 0 that represents a set of such conflicting candidates.
 * The caller might end up discarding such an entry anyway, but if it selects
 * such an entry it should react as though the call were ambiguous.
 *
 * If missing_ok is true, an empty list (NULL) is returned if the name was
 * schema-qualified with a schema that does not exist.  Likewise if no
 * candidate is found for other reasons.
 */


/*
 * MatchNamedCall
 *		Given a pg_proc heap tuple and a call's list of argument names,
 *		check whether the function could match the call.
 *
 * The call could match if all supplied argument names are accepted by
 * the function, in positions after the last positional argument, and there
 * are defaults for all unsupplied arguments.
 *
 * If include_out_arguments is true, we are treating OUT arguments as
 * included in the argument list.  pronargs is the number of arguments
 * we're considering (the length of either proargtypes or proallargtypes).
 *
 * The number of positional arguments is nargs - list_length(argnames).
 * Note caller has already done basic checks on argument count.
 *
 * On match, return true and fill *argnumbers with a palloc'd array showing
 * the mapping from call argument positions to actual function argument
 * numbers.  Defaulted arguments are included in this map, at positions
 * after the last supplied argument.
 */


/*
 * FunctionIsVisible
 *		Determine whether a function (identified by OID) is visible in the
 *		current search path.  Visible means "would be found by searching
 *		for the unqualified function name with exact argument matches".
 */


/*
 * FunctionIsVisibleExt
 *		As above, but if the function isn't found and is_missing is not NULL,
 *		then set *is_missing = true and return false instead of throwing
 *		an error.  (Caller must initialize *is_missing = false.)
 */



/*
 * OpernameGetOprid
 *		Given a possibly-qualified operator name and exact input datatypes,
 *		look up the operator.  Returns InvalidOid if not found.
 *
 * Pass oprleft = InvalidOid for a prefix op.
 *
 * If the operator name is not schema-qualified, it is sought in the current
 * namespace search path.  If the name is schema-qualified and the given
 * schema does not exist, InvalidOid is returned.
 */


/*
 * OpernameGetCandidates
 *		Given a possibly-qualified operator name and operator kind,
 *		retrieve a list of the possible matches.
 *
 * If oprkind is '\0', we return all operators matching the given name,
 * regardless of arguments.
 *
 * We search a single namespace if the operator name is qualified, else
 * all namespaces in the search path.  The return list will never contain
 * multiple entries with identical argument lists --- in the multiple-
 * namespace case, we arrange for entries in earlier namespaces to mask
 * identical entries in later namespaces.
 *
 * The returned items always have two args[] entries --- the first will be
 * InvalidOid for a prefix oprkind.  nargs is always 2, too.
 */
#define SPACE_PER_OP MAXALIGN(offsetof(struct _FuncCandidateList, args) + \
							  2 * sizeof(Oid))

/*
 * OperatorIsVisible
 *		Determine whether an operator (identified by OID) is visible in the
 *		current search path.  Visible means "would be found by searching
 *		for the unqualified operator name with exact argument matches".
 */


/*
 * OperatorIsVisibleExt
 *		As above, but if the operator isn't found and is_missing is not NULL,
 *		then set *is_missing = true and return false instead of throwing
 *		an error.  (Caller must initialize *is_missing = false.)
 */



/*
 * OpclassnameGetOpcid
 *		Try to resolve an unqualified index opclass name.
 *		Returns OID if opclass found in search path, else InvalidOid.
 *
 * This is essentially the same as TypenameGetTypid, but we have to have
 * an extra argument for the index AM OID.
 */


/*
 * OpclassIsVisible
 *		Determine whether an opclass (identified by OID) is visible in the
 *		current search path.  Visible means "would be found by searching
 *		for the unqualified opclass name".
 */


/*
 * OpclassIsVisibleExt
 *		As above, but if the opclass isn't found and is_missing is not NULL,
 *		then set *is_missing = true and return false instead of throwing
 *		an error.  (Caller must initialize *is_missing = false.)
 */


/*
 * OpfamilynameGetOpfid
 *		Try to resolve an unqualified index opfamily name.
 *		Returns OID if opfamily found in search path, else InvalidOid.
 *
 * This is essentially the same as TypenameGetTypid, but we have to have
 * an extra argument for the index AM OID.
 */


/*
 * OpfamilyIsVisible
 *		Determine whether an opfamily (identified by OID) is visible in the
 *		current search path.  Visible means "would be found by searching
 *		for the unqualified opfamily name".
 */


/*
 * OpfamilyIsVisibleExt
 *		As above, but if the opfamily isn't found and is_missing is not NULL,
 *		then set *is_missing = true and return false instead of throwing
 *		an error.  (Caller must initialize *is_missing = false.)
 */


/*
 * lookup_collation
 *		If there's a collation of the given name/namespace, and it works
 *		with the given encoding, return its OID.  Else return InvalidOid.
 */


/*
 * CollationGetCollid
 *		Try to resolve an unqualified collation name.
 *		Returns OID if collation found in search path, else InvalidOid.
 *
 * Note that this will only find collations that work with the current
 * database's encoding.
 */


/*
 * CollationIsVisible
 *		Determine whether a collation (identified by OID) is visible in the
 *		current search path.  Visible means "would be found by searching
 *		for the unqualified collation name".
 *
 * Note that only collations that work with the current database's encoding
 * will be considered visible.
 */


/*
 * CollationIsVisibleExt
 *		As above, but if the collation isn't found and is_missing is not NULL,
 *		then set *is_missing = true and return false instead of throwing
 *		an error.  (Caller must initialize *is_missing = false.)
 */



/*
 * ConversionGetConid
 *		Try to resolve an unqualified conversion name.
 *		Returns OID if conversion found in search path, else InvalidOid.
 *
 * This is essentially the same as RelnameGetRelid.
 */


/*
 * ConversionIsVisible
 *		Determine whether a conversion (identified by OID) is visible in the
 *		current search path.  Visible means "would be found by searching
 *		for the unqualified conversion name".
 */


/*
 * ConversionIsVisibleExt
 *		As above, but if the conversion isn't found and is_missing is not NULL,
 *		then set *is_missing = true and return false instead of throwing
 *		an error.  (Caller must initialize *is_missing = false.)
 */


/*
 * get_statistics_object_oid - find a statistics object by possibly qualified name
 *
 * If not found, returns InvalidOid if missing_ok, else throws error
 */


/*
 * StatisticsObjIsVisible
 *		Determine whether a statistics object (identified by OID) is visible in
 *		the current search path.  Visible means "would be found by searching
 *		for the unqualified statistics object name".
 */


/*
 * StatisticsObjIsVisibleExt
 *		As above, but if the statistics object isn't found and is_missing is
 *		not NULL, then set *is_missing = true and return false instead of
 *		throwing an error.  (Caller must initialize *is_missing = false.)
 */


/*
 * get_ts_parser_oid - find a TS parser by possibly qualified name
 *
 * If not found, returns InvalidOid if missing_ok, else throws error
 */


/*
 * TSParserIsVisible
 *		Determine whether a parser (identified by OID) is visible in the
 *		current search path.  Visible means "would be found by searching
 *		for the unqualified parser name".
 */


/*
 * TSParserIsVisibleExt
 *		As above, but if the parser isn't found and is_missing is not NULL,
 *		then set *is_missing = true and return false instead of throwing
 *		an error.  (Caller must initialize *is_missing = false.)
 */


/*
 * get_ts_dict_oid - find a TS dictionary by possibly qualified name
 *
 * If not found, returns InvalidOid if missing_ok, else throws error
 */


/*
 * TSDictionaryIsVisible
 *		Determine whether a dictionary (identified by OID) is visible in the
 *		current search path.  Visible means "would be found by searching
 *		for the unqualified dictionary name".
 */


/*
 * TSDictionaryIsVisibleExt
 *		As above, but if the dictionary isn't found and is_missing is not NULL,
 *		then set *is_missing = true and return false instead of throwing
 *		an error.  (Caller must initialize *is_missing = false.)
 */


/*
 * get_ts_template_oid - find a TS template by possibly qualified name
 *
 * If not found, returns InvalidOid if missing_ok, else throws error
 */


/*
 * TSTemplateIsVisible
 *		Determine whether a template (identified by OID) is visible in the
 *		current search path.  Visible means "would be found by searching
 *		for the unqualified template name".
 */


/*
 * TSTemplateIsVisibleExt
 *		As above, but if the template isn't found and is_missing is not NULL,
 *		then set *is_missing = true and return false instead of throwing
 *		an error.  (Caller must initialize *is_missing = false.)
 */


/*
 * get_ts_config_oid - find a TS config by possibly qualified name
 *
 * If not found, returns InvalidOid if missing_ok, else throws error
 */


/*
 * TSConfigIsVisible
 *		Determine whether a text search configuration (identified by OID)
 *		is visible in the current search path.  Visible means "would be found
 *		by searching for the unqualified text search configuration name".
 */


/*
 * TSConfigIsVisibleExt
 *		As above, but if the configuration isn't found and is_missing is not
 *		NULL, then set *is_missing = true and return false instead of throwing
 *		an error.  (Caller must initialize *is_missing = false.)
 */



/*
 * DeconstructQualifiedName
 *		Given a possibly-qualified name expressed as a list of String nodes,
 *		extract the schema name and object name.
 *
 * *nspname_p is set to NULL if there is no explicit schema name.
 */


/*
 * LookupNamespaceNoError
 *		Look up a schema name.
 *
 * Returns the namespace OID, or InvalidOid if not found.
 *
 * Note this does NOT perform any permissions check --- callers are
 * responsible for being sure that an appropriate check is made.
 * In the majority of cases LookupExplicitNamespace is preferable.
 */


/*
 * LookupExplicitNamespace
 *		Process an explicitly-specified schema name: look up the schema
 *		and verify we have USAGE (lookup) rights in it.
 *
 * Returns the namespace OID
 */


/*
 * LookupCreationNamespace
 *		Look up the schema and verify we have CREATE rights on it.
 *
 * This is just like LookupExplicitNamespace except for the different
 * permission check, and that we are willing to create pg_temp if needed.
 *
 * Note: calling this may result in a CommandCounterIncrement operation,
 * if we have to create or clean out the temp namespace.
 */


/*
 * Common checks on switching namespaces.
 *
 * We complain if either the old or new namespaces is a temporary schema
 * (or temporary toast schema), or if either the old or new namespaces is the
 * TOAST schema.
 */


/*
 * QualifiedNameGetCreationNamespace
 *		Given a possibly-qualified name for an object (in List-of-Strings
 *		format), determine what namespace the object should be created in.
 *		Also extract and return the object name (last component of list).
 *
 * Note: this does not apply any permissions check.  Callers must check
 * for CREATE rights on the selected namespace when appropriate.
 *
 * Note: calling this may result in a CommandCounterIncrement operation,
 * if we have to create or clean out the temp namespace.
 */


/*
 * get_namespace_oid - given a namespace name, look up the OID
 *
 * If missing_ok is false, throw an error if namespace name not found.  If
 * true, just return InvalidOid.
 */


/*
 * makeRangeVarFromNameList
 *		Utility routine to convert a qualified-name list into RangeVar form.
 */


/*
 * NameListToString
 *		Utility routine to convert a qualified-name list into a string.
 *
 * This is used primarily to form error messages, and so we do not quote
 * the list elements, for the sake of legibility.
 *
 * In most scenarios the list elements should always be String values,
 * but we also allow A_Star for the convenience of ColumnRef processing.
 */
char *
NameListToString(const List *names)
{
	StringInfoData string;
	ListCell   *l;

	initStringInfo(&string);

	foreach(l, names)
	{
		Node	   *name = (Node *) lfirst(l);

		if (l != list_head(names))
			appendStringInfoChar(&string, '.');

		if (IsA(name, String))
			appendStringInfoString(&string, strVal(name));
		else if (IsA(name, A_Star))
			appendStringInfoChar(&string, '*');
		else
			elog(ERROR, "unexpected node type in name list: %d",
				 (int) nodeTag(name));
	}

	return string.data;
}

/*
 * NameListToQuotedString
 *		Utility routine to convert a qualified-name list into a string.
 *
 * Same as above except that names will be double-quoted where necessary,
 * so the string could be re-parsed (eg, by textToQualifiedNameList).
 */


/*
 * isTempNamespace - is the given namespace my temporary-table namespace?
 */


/*
 * isTempToastNamespace - is the given namespace my temporary-toast-table
 *		namespace?
 */


/*
 * isTempOrTempToastNamespace - is the given namespace my temporary-table
 *		namespace or my temporary-toast-table namespace?
 */


/*
 * isAnyTempNamespace - is the given namespace a temporary-table namespace
 * (either my own, or another backend's)?  Temporary-toast-table namespaces
 * are included, too.
 */


/*
 * isOtherTempNamespace - is the given namespace some other backend's
 * temporary-table namespace (including temporary-toast-table namespaces)?
 *
 * Note: for most purposes in the C code, this function is obsolete.  Use
 * RELATION_IS_OTHER_TEMP() instead to detect non-local temp relations.
 */


/*
 * checkTempNamespaceStatus - is the given namespace owned and actively used
 * by a backend?
 *
 * Note: this can be used while scanning relations in pg_class to detect
 * orphaned temporary tables or namespaces with a backend connected to a
 * given database.  The result may be out of date quickly, so the caller
 * must be careful how to handle this information.
 */


/*
 * GetTempNamespaceProcNumber - if the given namespace is a temporary-table
 * namespace (either my own, or another backend's), return the proc number
 * that owns it.  Temporary-toast-table namespaces are included, too.
 * If it isn't a temp namespace, return INVALID_PROC_NUMBER.
 */


/*
 * GetTempToastNamespace - get the OID of my temporary-toast-table namespace,
 * which must already be assigned.  (This is only used when creating a toast
 * table for a temp table, so we must have already done InitTempTableNamespace)
 */



/*
 * GetTempNamespaceState - fetch status of session's temporary namespace
 *
 * This is used for conveying state to a parallel worker, and is not meant
 * for general-purpose access.
 */


/*
 * SetTempNamespaceState - set status of session's temporary namespace
 *
 * This is used for conveying state to a parallel worker, and is not meant for
 * general-purpose access.  By transferring these namespace OIDs to workers,
 * we ensure they will have the same notion of the search path as their leader
 * does.
 */



/*
 * GetSearchPathMatcher - fetch current search path definition.
 *
 * The result structure is allocated in the specified memory context
 * (which might or might not be equal to CurrentMemoryContext); but any
 * junk created by revalidation calculations will be in CurrentMemoryContext.
 */


/*
 * CopySearchPathMatcher - copy the specified SearchPathMatcher.
 *
 * The result structure is allocated in CurrentMemoryContext.
 */


/*
 * SearchPathMatchesCurrentEnvironment - does path match current environment?
 *
 * This is tested over and over in some common code paths, and in the typical
 * scenario where the active search path seldom changes, it'll always succeed.
 * We make that case fast by keeping a generation counter that is advanced
 * whenever the active search path changes.
 */


/*
 * get_collation_oid - find a collation by possibly qualified name
 *
 * Note that this will only find collations that work with the current
 * database's encoding.
 */
Oid get_collation_oid(List *name, bool missing_ok) { return DEFAULT_COLLATION_OID; }


/*
 * get_conversion_oid - find a conversion by possibly qualified name
 */


/*
 * FindDefaultConversionProc - find default encoding conversion proc
 */


/*
 * Look up namespace IDs and perform ACL checks. Return newly-allocated list.
 */


/*
 * Remove duplicates, run namespace search hooks, and prepend
 * implicitly-searched namespaces. Return newly-allocated list.
 *
 * If an object_access_hook is present, this must always be recalculated. It
 * may seem that duplicate elimination is not dependent on the result of the
 * hook, but if a hook returns different results on different calls for the
 * same namespace ID, then it could affect the order in which that namespace
 * appears in the final list.
 */


/*
 * Retrieve search path information from the cache; or if not there, fill
 * it. The returned entry is valid only until the next call to this function.
 */


/*
 * recomputeNamespacePath - recompute path derived variables if needed.
 */


/*
 * AccessTempTableNamespace
 *		Provide access to a temporary namespace, potentially creating it
 *		if not present yet.  This routine registers if the namespace gets
 *		in use in this transaction.  'force' can be set to true to allow
 *		the caller to enforce the creation of the temporary namespace for
 *		use in this backend, which happens if its creation is pending.
 */


/*
 * InitTempTableNamespace
 *		Initialize temp table namespace on first use in a particular backend
 */


/*
 * End-of-transaction cleanup for namespaces.
 */


/*
 * AtEOSubXact_Namespace
 *
 * At subtransaction commit, propagate the temp-namespace-creation
 * flag to the parent subtransaction.
 *
 * At subtransaction abort, forget the flag if we set it up.
 */


/*
 * Remove all relations in the specified temp namespace.
 *
 * This is called at backend shutdown (if we made any temp relations).
 * It is also called when we begin using a pre-existing temp namespace,
 * in order to clean out any relations that might have been created by
 * a crashed backend.
 */


/*
 * Callback to remove temp relations at backend exit.
 */


/*
 * Remove all temp tables from the temporary namespace.
 */



/*
 * Routines for handling the GUC variable 'search_path'.
 */

/* check_hook: validate new search_path value */


/* assign_hook: do extra actions as needed */


/*
 * InitializeSearchPath: initialize module during InitPostgres.
 *
 * This is called after we are up enough to be able to do catalog lookups.
 */


/*
 * InvalidationCallback
 *		Syscache inval callback function
 */


/*
 * Fetch the active search path. The return value is a palloc'ed list
 * of OIDs; the caller is responsible for freeing this storage as
 * appropriate.
 *
 * The returned list includes the implicitly-prepended namespaces only if
 * includeImplicit is true.
 *
 * Note: calling this may result in a CommandCounterIncrement operation,
 * if we have to create or clean out the temp namespace.
 */


/*
 * Fetch the active search path into a caller-allocated array of OIDs.
 * Returns the number of path entries.  (If this is more than sarray_len,
 * then the data didn't fit and is not all stored.)
 *
 * The returned list always includes the implicitly-prepended namespaces,
 * but never includes the temp namespace.  (This is suitable for existing
 * users, which would want to ignore the temp namespace anyway.)  This
 * definition allows us to not worry about initializing the temp namespace.
 */



/*
 * Export the FooIsVisible functions as SQL-callable functions.
 *
 * Note: as of Postgres 8.4, these will silently return NULL if called on
 * a nonexistent object OID, rather than failing.  This is to avoid race
 * condition errors when a query that's scanning a catalog using an MVCC
 * snapshot uses one of these functions.  The underlying IsVisible functions
 * always use an up-to-date snapshot and so might see the object as already
 * gone when it's still visible to the transaction snapshot.
 */






























