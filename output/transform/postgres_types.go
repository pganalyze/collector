package transform

import (
  snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
  "github.com/pganalyze/collector/state"
)

func transformPostgresTypes(s snapshot.FullSnapshot, transientState state.TransientState, databaseOidToIdx OidToIdx) (snapshot.FullSnapshot, OidToIdx) {
  typeOidToIdx := make(OidToIdx)

  for _, pgType := range transientState.Types {
    info := snapshot.TypeInformation{
      DatabaseIdx: databaseOidToIdx[pgType.DatabaseOid],
      SchemaName: pgType.SchemaName,
      Name: pgType.Name,
      Type: pgType.Type,
      UnderlyingType: pgType.UnderlyingType.String,
      NotNull: pgType.NotNull,
      Default: pgType.Default.String,
      Constraint: pgType.Constraint.String,
      EnumValues: pgType.EnumValues,
    }
    for _, attr := range pgType.CompositeAttrs {
      info.CompositeAttrs = append(info.CompositeAttrs, &snapshot.TypeInformation_CompositeAttr{Name: attr[0], Type: attr[1]})
    }

    idx := int32(len(s.TypeInformations))
    s.TypeInformations = append(s.TypeInformations, &info)
    typeOidToIdx[pgType.Oid] = idx
  }

  return s, typeOidToIdx
}
