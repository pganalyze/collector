package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformPostgresTypes(s snapshot.FullSnapshot, transientState state.TransientState, databaseOidToIdx OidToIdx) (snapshot.FullSnapshot, OidToIdx) {
	typeOidToIdx := make(OidToIdx)

	for _, pgType := range transientState.Types {
		var customType snapshot.CustomTypeInformation_Type
		switch pgType.Type {
		case "e":
			customType = snapshot.CustomTypeInformation_ENUM
		case "d":
			customType = snapshot.CustomTypeInformation_DOMAIN
		case "c":
			customType = snapshot.CustomTypeInformation_COMPOSITE
		case "b":
			customType = snapshot.CustomTypeInformation_BASE
		case "p":
			customType = snapshot.CustomTypeInformation_PSEUDO
		case "r":
			customType = snapshot.CustomTypeInformation_RANGE
		case "m":
			customType = snapshot.CustomTypeInformation_MULTIRANGE
		}

		info := snapshot.CustomTypeInformation{
			DatabaseIdx:       databaseOidToIdx[pgType.DatabaseOid],
			SchemaName:        pgType.SchemaName,
			Name:              pgType.Name,
			Type:              customType,
			DomainType:        pgType.DomainType.String,
			DomainNotNull:     pgType.DomainNotNull,
			DomainDefault:     pgType.DomainDefault.String,
			DomainConstraints: pgType.DomainConstraints,
			EnumValues:        pgType.EnumValues,
		}
		for _, attr := range pgType.CompositeAttrs {
			info.CompositeAttrs = append(info.CompositeAttrs, &snapshot.CustomTypeInformation_CompositeAttr{Name: attr[0], Type: attr[1]})
		}

		idx := int32(len(s.CustomTypeInformations))
		s.CustomTypeInformations = append(s.CustomTypeInformations, &info)
		typeOidToIdx[pgType.Oid] = idx
	}

	return s, typeOidToIdx
}
