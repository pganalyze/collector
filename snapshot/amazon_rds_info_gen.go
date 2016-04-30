package snapshot

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *AmazonRdsInfo) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var xvk uint32
	xvk, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for xvk > 0 {
		xvk--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "region":
			err = z.Region.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "instance_class":
			err = z.InstanceClass.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "instance_id":
			err = z.InstanceID.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "status":
			err = z.Status.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "availability_zone":
			err = z.AvailabilityZone.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "publicly_accessible":
			err = z.PubliclyAccessible.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "multi_az":
			err = z.MultiAZ.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "secondary_availability_zone":
			err = z.SecondaryAvailabilityZone.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "ca_certificate":
			err = z.CACertificate.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "auto_minor_version_upgrade":
			err = z.AutoMinorVersionUpgrade.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "preferred_maintenance_window":
			err = z.PreferredMaintenanceWindow.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "latest_restorable_time":
			err = z.LatestRestorableTime.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "preferred_backup_window":
			err = z.PreferredBackupWindow.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "backup_retention_period":
			err = z.BackupRetentionPeriod.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "master_username":
			err = z.MasterUsername.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "initial_db_name":
			err = z.InitialDbName.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "created_at":
			err = z.CreatedAt.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "storage_provisioned_iops":
			err = z.StorageProvisionedIops.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "storage_encrypted":
			err = z.StorageEncrypted.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "storage_type":
			err = z.StorageType.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "parameter_apply_status":
			z.ParameterApplyStatus, err = dc.ReadString()
			if err != nil {
				return
			}
		case "parameter_pgss_enabled":
			z.ParameterPgssEnabled, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "os_snapshot":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.OsSnapshot = nil
			} else {
				if z.OsSnapshot == nil {
					z.OsSnapshot = new(RdsOsSnapshot)
				}
				err = z.OsSnapshot.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *AmazonRdsInfo) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 23
	// write "region"
	err = en.Append(0xde, 0x0, 0x17, 0xa6, 0x72, 0x65, 0x67, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = z.Region.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "instance_class"
	err = en.Append(0xae, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x5f, 0x63, 0x6c, 0x61, 0x73, 0x73)
	if err != nil {
		return err
	}
	err = z.InstanceClass.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "instance_id"
	err = en.Append(0xab, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x5f, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = z.InstanceID.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "status"
	err = en.Append(0xa6, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73)
	if err != nil {
		return err
	}
	err = z.Status.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "availability_zone"
	err = en.Append(0xb1, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x5f, 0x7a, 0x6f, 0x6e, 0x65)
	if err != nil {
		return err
	}
	err = z.AvailabilityZone.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "publicly_accessible"
	err = en.Append(0xb3, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x6c, 0x79, 0x5f, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73, 0x69, 0x62, 0x6c, 0x65)
	if err != nil {
		return err
	}
	err = z.PubliclyAccessible.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "multi_az"
	err = en.Append(0xa8, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x5f, 0x61, 0x7a)
	if err != nil {
		return err
	}
	err = z.MultiAZ.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "secondary_availability_zone"
	err = en.Append(0xbb, 0x73, 0x65, 0x63, 0x6f, 0x6e, 0x64, 0x61, 0x72, 0x79, 0x5f, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x5f, 0x7a, 0x6f, 0x6e, 0x65)
	if err != nil {
		return err
	}
	err = z.SecondaryAvailabilityZone.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "ca_certificate"
	err = en.Append(0xae, 0x63, 0x61, 0x5f, 0x63, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65)
	if err != nil {
		return err
	}
	err = z.CACertificate.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "auto_minor_version_upgrade"
	err = en.Append(0xba, 0x61, 0x75, 0x74, 0x6f, 0x5f, 0x6d, 0x69, 0x6e, 0x6f, 0x72, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x5f, 0x75, 0x70, 0x67, 0x72, 0x61, 0x64, 0x65)
	if err != nil {
		return err
	}
	err = z.AutoMinorVersionUpgrade.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "preferred_maintenance_window"
	err = en.Append(0xbc, 0x70, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x65, 0x64, 0x5f, 0x6d, 0x61, 0x69, 0x6e, 0x74, 0x65, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x5f, 0x77, 0x69, 0x6e, 0x64, 0x6f, 0x77)
	if err != nil {
		return err
	}
	err = z.PreferredMaintenanceWindow.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "latest_restorable_time"
	err = en.Append(0xb6, 0x6c, 0x61, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x72, 0x65, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = z.LatestRestorableTime.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "preferred_backup_window"
	err = en.Append(0xb7, 0x70, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x65, 0x64, 0x5f, 0x62, 0x61, 0x63, 0x6b, 0x75, 0x70, 0x5f, 0x77, 0x69, 0x6e, 0x64, 0x6f, 0x77)
	if err != nil {
		return err
	}
	err = z.PreferredBackupWindow.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "backup_retention_period"
	err = en.Append(0xb7, 0x62, 0x61, 0x63, 0x6b, 0x75, 0x70, 0x5f, 0x72, 0x65, 0x74, 0x65, 0x6e, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x70, 0x65, 0x72, 0x69, 0x6f, 0x64)
	if err != nil {
		return err
	}
	err = z.BackupRetentionPeriod.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "master_username"
	err = en.Append(0xaf, 0x6d, 0x61, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = z.MasterUsername.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "initial_db_name"
	err = en.Append(0xaf, 0x69, 0x6e, 0x69, 0x74, 0x69, 0x61, 0x6c, 0x5f, 0x64, 0x62, 0x5f, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = z.InitialDbName.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "created_at"
	err = en.Append(0xaa, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74)
	if err != nil {
		return err
	}
	err = z.CreatedAt.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "storage_provisioned_iops"
	err = en.Append(0xb8, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x5f, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x65, 0x64, 0x5f, 0x69, 0x6f, 0x70, 0x73)
	if err != nil {
		return err
	}
	err = z.StorageProvisionedIops.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "storage_encrypted"
	err = en.Append(0xb1, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x5f, 0x65, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = z.StorageEncrypted.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "storage_type"
	err = en.Append(0xac, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x5f, 0x74, 0x79, 0x70, 0x65)
	if err != nil {
		return err
	}
	err = z.StorageType.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "parameter_apply_status"
	err = en.Append(0xb6, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x5f, 0x61, 0x70, 0x70, 0x6c, 0x79, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteString(z.ParameterApplyStatus)
	if err != nil {
		return
	}
	// write "parameter_pgss_enabled"
	err = en.Append(0xb6, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x5f, 0x70, 0x67, 0x73, 0x73, 0x5f, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.ParameterPgssEnabled)
	if err != nil {
		return
	}
	// write "os_snapshot"
	err = en.Append(0xab, 0x6f, 0x73, 0x5f, 0x73, 0x6e, 0x61, 0x70, 0x73, 0x68, 0x6f, 0x74)
	if err != nil {
		return err
	}
	if z.OsSnapshot == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.OsSnapshot.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *AmazonRdsInfo) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 23
	// string "region"
	o = append(o, 0xde, 0x0, 0x17, 0xa6, 0x72, 0x65, 0x67, 0x69, 0x6f, 0x6e)
	o, err = z.Region.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "instance_class"
	o = append(o, 0xae, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x5f, 0x63, 0x6c, 0x61, 0x73, 0x73)
	o, err = z.InstanceClass.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "instance_id"
	o = append(o, 0xab, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x5f, 0x69, 0x64)
	o, err = z.InstanceID.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "status"
	o = append(o, 0xa6, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73)
	o, err = z.Status.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "availability_zone"
	o = append(o, 0xb1, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x5f, 0x7a, 0x6f, 0x6e, 0x65)
	o, err = z.AvailabilityZone.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "publicly_accessible"
	o = append(o, 0xb3, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x6c, 0x79, 0x5f, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73, 0x69, 0x62, 0x6c, 0x65)
	o, err = z.PubliclyAccessible.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "multi_az"
	o = append(o, 0xa8, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x5f, 0x61, 0x7a)
	o, err = z.MultiAZ.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "secondary_availability_zone"
	o = append(o, 0xbb, 0x73, 0x65, 0x63, 0x6f, 0x6e, 0x64, 0x61, 0x72, 0x79, 0x5f, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x5f, 0x7a, 0x6f, 0x6e, 0x65)
	o, err = z.SecondaryAvailabilityZone.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "ca_certificate"
	o = append(o, 0xae, 0x63, 0x61, 0x5f, 0x63, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65)
	o, err = z.CACertificate.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "auto_minor_version_upgrade"
	o = append(o, 0xba, 0x61, 0x75, 0x74, 0x6f, 0x5f, 0x6d, 0x69, 0x6e, 0x6f, 0x72, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x5f, 0x75, 0x70, 0x67, 0x72, 0x61, 0x64, 0x65)
	o, err = z.AutoMinorVersionUpgrade.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "preferred_maintenance_window"
	o = append(o, 0xbc, 0x70, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x65, 0x64, 0x5f, 0x6d, 0x61, 0x69, 0x6e, 0x74, 0x65, 0x6e, 0x61, 0x6e, 0x63, 0x65, 0x5f, 0x77, 0x69, 0x6e, 0x64, 0x6f, 0x77)
	o, err = z.PreferredMaintenanceWindow.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "latest_restorable_time"
	o = append(o, 0xb6, 0x6c, 0x61, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x72, 0x65, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65)
	o, err = z.LatestRestorableTime.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "preferred_backup_window"
	o = append(o, 0xb7, 0x70, 0x72, 0x65, 0x66, 0x65, 0x72, 0x72, 0x65, 0x64, 0x5f, 0x62, 0x61, 0x63, 0x6b, 0x75, 0x70, 0x5f, 0x77, 0x69, 0x6e, 0x64, 0x6f, 0x77)
	o, err = z.PreferredBackupWindow.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "backup_retention_period"
	o = append(o, 0xb7, 0x62, 0x61, 0x63, 0x6b, 0x75, 0x70, 0x5f, 0x72, 0x65, 0x74, 0x65, 0x6e, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x70, 0x65, 0x72, 0x69, 0x6f, 0x64)
	o, err = z.BackupRetentionPeriod.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "master_username"
	o = append(o, 0xaf, 0x6d, 0x61, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65)
	o, err = z.MasterUsername.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "initial_db_name"
	o = append(o, 0xaf, 0x69, 0x6e, 0x69, 0x74, 0x69, 0x61, 0x6c, 0x5f, 0x64, 0x62, 0x5f, 0x6e, 0x61, 0x6d, 0x65)
	o, err = z.InitialDbName.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "created_at"
	o = append(o, 0xaa, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74)
	o, err = z.CreatedAt.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "storage_provisioned_iops"
	o = append(o, 0xb8, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x5f, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x65, 0x64, 0x5f, 0x69, 0x6f, 0x70, 0x73)
	o, err = z.StorageProvisionedIops.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "storage_encrypted"
	o = append(o, 0xb1, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x5f, 0x65, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x65, 0x64)
	o, err = z.StorageEncrypted.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "storage_type"
	o = append(o, 0xac, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x5f, 0x74, 0x79, 0x70, 0x65)
	o, err = z.StorageType.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "parameter_apply_status"
	o = append(o, 0xb6, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x5f, 0x61, 0x70, 0x70, 0x6c, 0x79, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73)
	o = msgp.AppendString(o, z.ParameterApplyStatus)
	// string "parameter_pgss_enabled"
	o = append(o, 0xb6, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x5f, 0x70, 0x67, 0x73, 0x73, 0x5f, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64)
	o = msgp.AppendBool(o, z.ParameterPgssEnabled)
	// string "os_snapshot"
	o = append(o, 0xab, 0x6f, 0x73, 0x5f, 0x73, 0x6e, 0x61, 0x70, 0x73, 0x68, 0x6f, 0x74)
	if z.OsSnapshot == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.OsSnapshot.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *AmazonRdsInfo) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var bzg uint32
	bzg, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for bzg > 0 {
		bzg--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "region":
			bts, err = z.Region.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "instance_class":
			bts, err = z.InstanceClass.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "instance_id":
			bts, err = z.InstanceID.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "status":
			bts, err = z.Status.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "availability_zone":
			bts, err = z.AvailabilityZone.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "publicly_accessible":
			bts, err = z.PubliclyAccessible.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "multi_az":
			bts, err = z.MultiAZ.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "secondary_availability_zone":
			bts, err = z.SecondaryAvailabilityZone.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "ca_certificate":
			bts, err = z.CACertificate.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "auto_minor_version_upgrade":
			bts, err = z.AutoMinorVersionUpgrade.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "preferred_maintenance_window":
			bts, err = z.PreferredMaintenanceWindow.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "latest_restorable_time":
			bts, err = z.LatestRestorableTime.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "preferred_backup_window":
			bts, err = z.PreferredBackupWindow.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "backup_retention_period":
			bts, err = z.BackupRetentionPeriod.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "master_username":
			bts, err = z.MasterUsername.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "initial_db_name":
			bts, err = z.InitialDbName.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "created_at":
			bts, err = z.CreatedAt.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "storage_provisioned_iops":
			bts, err = z.StorageProvisionedIops.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "storage_encrypted":
			bts, err = z.StorageEncrypted.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "storage_type":
			bts, err = z.StorageType.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "parameter_apply_status":
			z.ParameterApplyStatus, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "parameter_pgss_enabled":
			z.ParameterPgssEnabled, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "os_snapshot":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.OsSnapshot = nil
			} else {
				if z.OsSnapshot == nil {
					z.OsSnapshot = new(RdsOsSnapshot)
				}
				bts, err = z.OsSnapshot.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z *AmazonRdsInfo) Msgsize() (s int) {
	s = 3 + 7 + z.Region.Msgsize() + 15 + z.InstanceClass.Msgsize() + 12 + z.InstanceID.Msgsize() + 7 + z.Status.Msgsize() + 18 + z.AvailabilityZone.Msgsize() + 20 + z.PubliclyAccessible.Msgsize() + 9 + z.MultiAZ.Msgsize() + 28 + z.SecondaryAvailabilityZone.Msgsize() + 15 + z.CACertificate.Msgsize() + 27 + z.AutoMinorVersionUpgrade.Msgsize() + 29 + z.PreferredMaintenanceWindow.Msgsize() + 23 + z.LatestRestorableTime.Msgsize() + 24 + z.PreferredBackupWindow.Msgsize() + 24 + z.BackupRetentionPeriod.Msgsize() + 16 + z.MasterUsername.Msgsize() + 16 + z.InitialDbName.Msgsize() + 11 + z.CreatedAt.Msgsize() + 25 + z.StorageProvisionedIops.Msgsize() + 18 + z.StorageEncrypted.Msgsize() + 13 + z.StorageType.Msgsize() + 23 + msgp.StringPrefixSize + len(z.ParameterApplyStatus) + 23 + msgp.BoolSize + 12
	if z.OsSnapshot == nil {
		s += msgp.NilSize
	} else {
		s += z.OsSnapshot.Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RdsOsCPUUtilization) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var bai uint32
	bai, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for bai > 0 {
		bai--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "guest":
			z.Guest, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "irq":
			z.Irq, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "system":
			z.System, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "wait":
			z.Wait, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "idle":
			z.Idle, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "user":
			z.User, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "total":
			z.Total, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "steal":
			z.Steal, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "nice":
			z.Nice, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RdsOsCPUUtilization) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 9
	// write "guest"
	err = en.Append(0x89, 0xa5, 0x67, 0x75, 0x65, 0x73, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.Guest)
	if err != nil {
		return
	}
	// write "irq"
	err = en.Append(0xa3, 0x69, 0x72, 0x71)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.Irq)
	if err != nil {
		return
	}
	// write "system"
	err = en.Append(0xa6, 0x73, 0x79, 0x73, 0x74, 0x65, 0x6d)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.System)
	if err != nil {
		return
	}
	// write "wait"
	err = en.Append(0xa4, 0x77, 0x61, 0x69, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.Wait)
	if err != nil {
		return
	}
	// write "idle"
	err = en.Append(0xa4, 0x69, 0x64, 0x6c, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.Idle)
	if err != nil {
		return
	}
	// write "user"
	err = en.Append(0xa4, 0x75, 0x73, 0x65, 0x72)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.User)
	if err != nil {
		return
	}
	// write "total"
	err = en.Append(0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.Total)
	if err != nil {
		return
	}
	// write "steal"
	err = en.Append(0xa5, 0x73, 0x74, 0x65, 0x61, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.Steal)
	if err != nil {
		return
	}
	// write "nice"
	err = en.Append(0xa4, 0x6e, 0x69, 0x63, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.Nice)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RdsOsCPUUtilization) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 9
	// string "guest"
	o = append(o, 0x89, 0xa5, 0x67, 0x75, 0x65, 0x73, 0x74)
	o = msgp.AppendFloat32(o, z.Guest)
	// string "irq"
	o = append(o, 0xa3, 0x69, 0x72, 0x71)
	o = msgp.AppendFloat32(o, z.Irq)
	// string "system"
	o = append(o, 0xa6, 0x73, 0x79, 0x73, 0x74, 0x65, 0x6d)
	o = msgp.AppendFloat32(o, z.System)
	// string "wait"
	o = append(o, 0xa4, 0x77, 0x61, 0x69, 0x74)
	o = msgp.AppendFloat32(o, z.Wait)
	// string "idle"
	o = append(o, 0xa4, 0x69, 0x64, 0x6c, 0x65)
	o = msgp.AppendFloat32(o, z.Idle)
	// string "user"
	o = append(o, 0xa4, 0x75, 0x73, 0x65, 0x72)
	o = msgp.AppendFloat32(o, z.User)
	// string "total"
	o = append(o, 0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	o = msgp.AppendFloat32(o, z.Total)
	// string "steal"
	o = append(o, 0xa5, 0x73, 0x74, 0x65, 0x61, 0x6c)
	o = msgp.AppendFloat32(o, z.Steal)
	// string "nice"
	o = append(o, 0xa4, 0x6e, 0x69, 0x63, 0x65)
	o = msgp.AppendFloat32(o, z.Nice)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RdsOsCPUUtilization) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var cmr uint32
	cmr, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for cmr > 0 {
		cmr--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "guest":
			z.Guest, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "irq":
			z.Irq, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "system":
			z.System, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "wait":
			z.Wait, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "idle":
			z.Idle, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "user":
			z.User, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "total":
			z.Total, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "steal":
			z.Steal, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "nice":
			z.Nice, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z *RdsOsCPUUtilization) Msgsize() (s int) {
	s = 1 + 6 + msgp.Float32Size + 4 + msgp.Float32Size + 7 + msgp.Float32Size + 5 + msgp.Float32Size + 5 + msgp.Float32Size + 5 + msgp.Float32Size + 6 + msgp.Float32Size + 6 + msgp.Float32Size + 5 + msgp.Float32Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RdsOsDiskIO) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var ajw uint32
	ajw, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for ajw > 0 {
		ajw--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "writeKbPS":
			z.WriteKbPS, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "readIOsPS":
			z.ReadIOsPS, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "await":
			z.Await, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "readKbPS":
			z.ReadKbPS, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "rrqmPS":
			z.RrqmPS, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "util":
			z.Util, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "avgQueueLen":
			z.AvgQueueLen, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "tps":
			z.Tps, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "readKb":
			z.ReadKb, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "device":
			z.Device, err = dc.ReadString()
			if err != nil {
				return
			}
		case "writeKb":
			z.WriteKb, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "avgReqSz":
			z.AvgReqSz, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "wrqmPS":
			z.WrqmPS, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "writeIOsPS":
			z.WriteIOsPS, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RdsOsDiskIO) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 14
	// write "writeKbPS"
	err = en.Append(0x8e, 0xa9, 0x77, 0x72, 0x69, 0x74, 0x65, 0x4b, 0x62, 0x50, 0x53)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.WriteKbPS)
	if err != nil {
		return
	}
	// write "readIOsPS"
	err = en.Append(0xa9, 0x72, 0x65, 0x61, 0x64, 0x49, 0x4f, 0x73, 0x50, 0x53)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.ReadIOsPS)
	if err != nil {
		return
	}
	// write "await"
	err = en.Append(0xa5, 0x61, 0x77, 0x61, 0x69, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.Await)
	if err != nil {
		return
	}
	// write "readKbPS"
	err = en.Append(0xa8, 0x72, 0x65, 0x61, 0x64, 0x4b, 0x62, 0x50, 0x53)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.ReadKbPS)
	if err != nil {
		return
	}
	// write "rrqmPS"
	err = en.Append(0xa6, 0x72, 0x72, 0x71, 0x6d, 0x50, 0x53)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.RrqmPS)
	if err != nil {
		return
	}
	// write "util"
	err = en.Append(0xa4, 0x75, 0x74, 0x69, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.Util)
	if err != nil {
		return
	}
	// write "avgQueueLen"
	err = en.Append(0xab, 0x61, 0x76, 0x67, 0x51, 0x75, 0x65, 0x75, 0x65, 0x4c, 0x65, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.AvgQueueLen)
	if err != nil {
		return
	}
	// write "tps"
	err = en.Append(0xa3, 0x74, 0x70, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.Tps)
	if err != nil {
		return
	}
	// write "readKb"
	err = en.Append(0xa6, 0x72, 0x65, 0x61, 0x64, 0x4b, 0x62)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.ReadKb)
	if err != nil {
		return
	}
	// write "device"
	err = en.Append(0xa6, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Device)
	if err != nil {
		return
	}
	// write "writeKb"
	err = en.Append(0xa7, 0x77, 0x72, 0x69, 0x74, 0x65, 0x4b, 0x62)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.WriteKb)
	if err != nil {
		return
	}
	// write "avgReqSz"
	err = en.Append(0xa8, 0x61, 0x76, 0x67, 0x52, 0x65, 0x71, 0x53, 0x7a)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.AvgReqSz)
	if err != nil {
		return
	}
	// write "wrqmPS"
	err = en.Append(0xa6, 0x77, 0x72, 0x71, 0x6d, 0x50, 0x53)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.WrqmPS)
	if err != nil {
		return
	}
	// write "writeIOsPS"
	err = en.Append(0xaa, 0x77, 0x72, 0x69, 0x74, 0x65, 0x49, 0x4f, 0x73, 0x50, 0x53)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.WriteIOsPS)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RdsOsDiskIO) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 14
	// string "writeKbPS"
	o = append(o, 0x8e, 0xa9, 0x77, 0x72, 0x69, 0x74, 0x65, 0x4b, 0x62, 0x50, 0x53)
	o = msgp.AppendFloat32(o, z.WriteKbPS)
	// string "readIOsPS"
	o = append(o, 0xa9, 0x72, 0x65, 0x61, 0x64, 0x49, 0x4f, 0x73, 0x50, 0x53)
	o = msgp.AppendFloat32(o, z.ReadIOsPS)
	// string "await"
	o = append(o, 0xa5, 0x61, 0x77, 0x61, 0x69, 0x74)
	o = msgp.AppendFloat32(o, z.Await)
	// string "readKbPS"
	o = append(o, 0xa8, 0x72, 0x65, 0x61, 0x64, 0x4b, 0x62, 0x50, 0x53)
	o = msgp.AppendFloat32(o, z.ReadKbPS)
	// string "rrqmPS"
	o = append(o, 0xa6, 0x72, 0x72, 0x71, 0x6d, 0x50, 0x53)
	o = msgp.AppendFloat32(o, z.RrqmPS)
	// string "util"
	o = append(o, 0xa4, 0x75, 0x74, 0x69, 0x6c)
	o = msgp.AppendFloat32(o, z.Util)
	// string "avgQueueLen"
	o = append(o, 0xab, 0x61, 0x76, 0x67, 0x51, 0x75, 0x65, 0x75, 0x65, 0x4c, 0x65, 0x6e)
	o = msgp.AppendFloat32(o, z.AvgQueueLen)
	// string "tps"
	o = append(o, 0xa3, 0x74, 0x70, 0x73)
	o = msgp.AppendFloat32(o, z.Tps)
	// string "readKb"
	o = append(o, 0xa6, 0x72, 0x65, 0x61, 0x64, 0x4b, 0x62)
	o = msgp.AppendFloat32(o, z.ReadKb)
	// string "device"
	o = append(o, 0xa6, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65)
	o = msgp.AppendString(o, z.Device)
	// string "writeKb"
	o = append(o, 0xa7, 0x77, 0x72, 0x69, 0x74, 0x65, 0x4b, 0x62)
	o = msgp.AppendFloat32(o, z.WriteKb)
	// string "avgReqSz"
	o = append(o, 0xa8, 0x61, 0x76, 0x67, 0x52, 0x65, 0x71, 0x53, 0x7a)
	o = msgp.AppendFloat32(o, z.AvgReqSz)
	// string "wrqmPS"
	o = append(o, 0xa6, 0x77, 0x72, 0x71, 0x6d, 0x50, 0x53)
	o = msgp.AppendFloat32(o, z.WrqmPS)
	// string "writeIOsPS"
	o = append(o, 0xaa, 0x77, 0x72, 0x69, 0x74, 0x65, 0x49, 0x4f, 0x73, 0x50, 0x53)
	o = msgp.AppendFloat32(o, z.WriteIOsPS)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RdsOsDiskIO) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var wht uint32
	wht, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for wht > 0 {
		wht--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "writeKbPS":
			z.WriteKbPS, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "readIOsPS":
			z.ReadIOsPS, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "await":
			z.Await, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "readKbPS":
			z.ReadKbPS, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "rrqmPS":
			z.RrqmPS, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "util":
			z.Util, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "avgQueueLen":
			z.AvgQueueLen, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "tps":
			z.Tps, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "readKb":
			z.ReadKb, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "device":
			z.Device, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "writeKb":
			z.WriteKb, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "avgReqSz":
			z.AvgReqSz, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "wrqmPS":
			z.WrqmPS, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "writeIOsPS":
			z.WriteIOsPS, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z *RdsOsDiskIO) Msgsize() (s int) {
	s = 1 + 10 + msgp.Float32Size + 10 + msgp.Float32Size + 6 + msgp.Float32Size + 9 + msgp.Float32Size + 7 + msgp.Float32Size + 5 + msgp.Float32Size + 12 + msgp.Float32Size + 4 + msgp.Float32Size + 7 + msgp.Float32Size + 7 + msgp.StringPrefixSize + len(z.Device) + 8 + msgp.Float32Size + 9 + msgp.Float32Size + 7 + msgp.Float32Size + 11 + msgp.Float32Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RdsOsFileSystem) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var hct uint32
	hct, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for hct > 0 {
		hct--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "used":
			z.Used, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "name":
			z.Name, err = dc.ReadString()
			if err != nil {
				return
			}
		case "usedFiles":
			z.UsedFiles, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "usedFilePercent":
			z.UsedFilePercent, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "maxFiles":
			z.MaxFiles, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "mountPoint":
			z.MountPoint, err = dc.ReadString()
			if err != nil {
				return
			}
		case "total":
			z.Total, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "usedPercent":
			z.UsedPercent, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RdsOsFileSystem) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 8
	// write "used"
	err = en.Append(0x88, 0xa4, 0x75, 0x73, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Used)
	if err != nil {
		return
	}
	// write "name"
	err = en.Append(0xa4, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Name)
	if err != nil {
		return
	}
	// write "usedFiles"
	err = en.Append(0xa9, 0x75, 0x73, 0x65, 0x64, 0x46, 0x69, 0x6c, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.UsedFiles)
	if err != nil {
		return
	}
	// write "usedFilePercent"
	err = en.Append(0xaf, 0x75, 0x73, 0x65, 0x64, 0x46, 0x69, 0x6c, 0x65, 0x50, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.UsedFilePercent)
	if err != nil {
		return
	}
	// write "maxFiles"
	err = en.Append(0xa8, 0x6d, 0x61, 0x78, 0x46, 0x69, 0x6c, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.MaxFiles)
	if err != nil {
		return
	}
	// write "mountPoint"
	err = en.Append(0xaa, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x50, 0x6f, 0x69, 0x6e, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.MountPoint)
	if err != nil {
		return
	}
	// write "total"
	err = en.Append(0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Total)
	if err != nil {
		return
	}
	// write "usedPercent"
	err = en.Append(0xab, 0x75, 0x73, 0x65, 0x64, 0x50, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.UsedPercent)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RdsOsFileSystem) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 8
	// string "used"
	o = append(o, 0x88, 0xa4, 0x75, 0x73, 0x65, 0x64)
	o = msgp.AppendInt64(o, z.Used)
	// string "name"
	o = append(o, 0xa4, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.Name)
	// string "usedFiles"
	o = append(o, 0xa9, 0x75, 0x73, 0x65, 0x64, 0x46, 0x69, 0x6c, 0x65, 0x73)
	o = msgp.AppendInt64(o, z.UsedFiles)
	// string "usedFilePercent"
	o = append(o, 0xaf, 0x75, 0x73, 0x65, 0x64, 0x46, 0x69, 0x6c, 0x65, 0x50, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74)
	o = msgp.AppendFloat32(o, z.UsedFilePercent)
	// string "maxFiles"
	o = append(o, 0xa8, 0x6d, 0x61, 0x78, 0x46, 0x69, 0x6c, 0x65, 0x73)
	o = msgp.AppendInt64(o, z.MaxFiles)
	// string "mountPoint"
	o = append(o, 0xaa, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x50, 0x6f, 0x69, 0x6e, 0x74)
	o = msgp.AppendString(o, z.MountPoint)
	// string "total"
	o = append(o, 0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	o = msgp.AppendInt64(o, z.Total)
	// string "usedPercent"
	o = append(o, 0xab, 0x75, 0x73, 0x65, 0x64, 0x50, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74)
	o = msgp.AppendFloat32(o, z.UsedPercent)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RdsOsFileSystem) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var cua uint32
	cua, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for cua > 0 {
		cua--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "used":
			z.Used, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "name":
			z.Name, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "usedFiles":
			z.UsedFiles, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "usedFilePercent":
			z.UsedFilePercent, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "maxFiles":
			z.MaxFiles, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "mountPoint":
			z.MountPoint, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "total":
			z.Total, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "usedPercent":
			z.UsedPercent, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z *RdsOsFileSystem) Msgsize() (s int) {
	s = 1 + 5 + msgp.Int64Size + 5 + msgp.StringPrefixSize + len(z.Name) + 10 + msgp.Int64Size + 16 + msgp.Float32Size + 9 + msgp.Int64Size + 11 + msgp.StringPrefixSize + len(z.MountPoint) + 6 + msgp.Int64Size + 12 + msgp.Float32Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RdsOsLoadAverageMinute) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var xhx uint32
	xhx, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for xhx > 0 {
		xhx--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "fifteen":
			z.Fifteen, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "five":
			z.Five, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "one":
			z.One, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z RdsOsLoadAverageMinute) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "fifteen"
	err = en.Append(0x83, 0xa7, 0x66, 0x69, 0x66, 0x74, 0x65, 0x65, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.Fifteen)
	if err != nil {
		return
	}
	// write "five"
	err = en.Append(0xa4, 0x66, 0x69, 0x76, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.Five)
	if err != nil {
		return
	}
	// write "one"
	err = en.Append(0xa3, 0x6f, 0x6e, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.One)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z RdsOsLoadAverageMinute) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "fifteen"
	o = append(o, 0x83, 0xa7, 0x66, 0x69, 0x66, 0x74, 0x65, 0x65, 0x6e)
	o = msgp.AppendFloat32(o, z.Fifteen)
	// string "five"
	o = append(o, 0xa4, 0x66, 0x69, 0x76, 0x65)
	o = msgp.AppendFloat32(o, z.Five)
	// string "one"
	o = append(o, 0xa3, 0x6f, 0x6e, 0x65)
	o = msgp.AppendFloat32(o, z.One)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RdsOsLoadAverageMinute) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var lqf uint32
	lqf, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for lqf > 0 {
		lqf--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "fifteen":
			z.Fifteen, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "five":
			z.Five, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "one":
			z.One, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z RdsOsLoadAverageMinute) Msgsize() (s int) {
	s = 1 + 8 + msgp.Float32Size + 5 + msgp.Float32Size + 4 + msgp.Float32Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RdsOsMemory) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var daf uint32
	daf, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for daf > 0 {
		daf--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "writeback":
			z.Writeback, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "hugePagesFree":
			z.HugePagesFree, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "hugePagesRsvd":
			z.HugePagesRsvd, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "hugePagesSurp":
			z.HugePagesSurp, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "cached":
			z.Cached, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "hugePagesSize":
			z.HugePagesSize, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "free":
			z.Free, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "hugePagesTotal":
			z.HugePagesTotal, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "inactive":
			z.Inactive, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "pageTables":
			z.PageTables, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "dirty":
			z.Dirty, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "mapped":
			z.Mapped, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "active":
			z.Active, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "total":
			z.Total, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "slab":
			z.Slab, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "buffers":
			z.Buffers, err = dc.ReadInt64()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RdsOsMemory) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 16
	// write "writeback"
	err = en.Append(0xde, 0x0, 0x10, 0xa9, 0x77, 0x72, 0x69, 0x74, 0x65, 0x62, 0x61, 0x63, 0x6b)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Writeback)
	if err != nil {
		return
	}
	// write "hugePagesFree"
	err = en.Append(0xad, 0x68, 0x75, 0x67, 0x65, 0x50, 0x61, 0x67, 0x65, 0x73, 0x46, 0x72, 0x65, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.HugePagesFree)
	if err != nil {
		return
	}
	// write "hugePagesRsvd"
	err = en.Append(0xad, 0x68, 0x75, 0x67, 0x65, 0x50, 0x61, 0x67, 0x65, 0x73, 0x52, 0x73, 0x76, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.HugePagesRsvd)
	if err != nil {
		return
	}
	// write "hugePagesSurp"
	err = en.Append(0xad, 0x68, 0x75, 0x67, 0x65, 0x50, 0x61, 0x67, 0x65, 0x73, 0x53, 0x75, 0x72, 0x70)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.HugePagesSurp)
	if err != nil {
		return
	}
	// write "cached"
	err = en.Append(0xa6, 0x63, 0x61, 0x63, 0x68, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Cached)
	if err != nil {
		return
	}
	// write "hugePagesSize"
	err = en.Append(0xad, 0x68, 0x75, 0x67, 0x65, 0x50, 0x61, 0x67, 0x65, 0x73, 0x53, 0x69, 0x7a, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.HugePagesSize)
	if err != nil {
		return
	}
	// write "free"
	err = en.Append(0xa4, 0x66, 0x72, 0x65, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Free)
	if err != nil {
		return
	}
	// write "hugePagesTotal"
	err = en.Append(0xae, 0x68, 0x75, 0x67, 0x65, 0x50, 0x61, 0x67, 0x65, 0x73, 0x54, 0x6f, 0x74, 0x61, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.HugePagesTotal)
	if err != nil {
		return
	}
	// write "inactive"
	err = en.Append(0xa8, 0x69, 0x6e, 0x61, 0x63, 0x74, 0x69, 0x76, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Inactive)
	if err != nil {
		return
	}
	// write "pageTables"
	err = en.Append(0xaa, 0x70, 0x61, 0x67, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.PageTables)
	if err != nil {
		return
	}
	// write "dirty"
	err = en.Append(0xa5, 0x64, 0x69, 0x72, 0x74, 0x79)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Dirty)
	if err != nil {
		return
	}
	// write "mapped"
	err = en.Append(0xa6, 0x6d, 0x61, 0x70, 0x70, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Mapped)
	if err != nil {
		return
	}
	// write "active"
	err = en.Append(0xa6, 0x61, 0x63, 0x74, 0x69, 0x76, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Active)
	if err != nil {
		return
	}
	// write "total"
	err = en.Append(0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Total)
	if err != nil {
		return
	}
	// write "slab"
	err = en.Append(0xa4, 0x73, 0x6c, 0x61, 0x62)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Slab)
	if err != nil {
		return
	}
	// write "buffers"
	err = en.Append(0xa7, 0x62, 0x75, 0x66, 0x66, 0x65, 0x72, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Buffers)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RdsOsMemory) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 16
	// string "writeback"
	o = append(o, 0xde, 0x0, 0x10, 0xa9, 0x77, 0x72, 0x69, 0x74, 0x65, 0x62, 0x61, 0x63, 0x6b)
	o = msgp.AppendInt64(o, z.Writeback)
	// string "hugePagesFree"
	o = append(o, 0xad, 0x68, 0x75, 0x67, 0x65, 0x50, 0x61, 0x67, 0x65, 0x73, 0x46, 0x72, 0x65, 0x65)
	o = msgp.AppendInt64(o, z.HugePagesFree)
	// string "hugePagesRsvd"
	o = append(o, 0xad, 0x68, 0x75, 0x67, 0x65, 0x50, 0x61, 0x67, 0x65, 0x73, 0x52, 0x73, 0x76, 0x64)
	o = msgp.AppendInt64(o, z.HugePagesRsvd)
	// string "hugePagesSurp"
	o = append(o, 0xad, 0x68, 0x75, 0x67, 0x65, 0x50, 0x61, 0x67, 0x65, 0x73, 0x53, 0x75, 0x72, 0x70)
	o = msgp.AppendInt64(o, z.HugePagesSurp)
	// string "cached"
	o = append(o, 0xa6, 0x63, 0x61, 0x63, 0x68, 0x65, 0x64)
	o = msgp.AppendInt64(o, z.Cached)
	// string "hugePagesSize"
	o = append(o, 0xad, 0x68, 0x75, 0x67, 0x65, 0x50, 0x61, 0x67, 0x65, 0x73, 0x53, 0x69, 0x7a, 0x65)
	o = msgp.AppendInt64(o, z.HugePagesSize)
	// string "free"
	o = append(o, 0xa4, 0x66, 0x72, 0x65, 0x65)
	o = msgp.AppendInt64(o, z.Free)
	// string "hugePagesTotal"
	o = append(o, 0xae, 0x68, 0x75, 0x67, 0x65, 0x50, 0x61, 0x67, 0x65, 0x73, 0x54, 0x6f, 0x74, 0x61, 0x6c)
	o = msgp.AppendInt64(o, z.HugePagesTotal)
	// string "inactive"
	o = append(o, 0xa8, 0x69, 0x6e, 0x61, 0x63, 0x74, 0x69, 0x76, 0x65)
	o = msgp.AppendInt64(o, z.Inactive)
	// string "pageTables"
	o = append(o, 0xaa, 0x70, 0x61, 0x67, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x73)
	o = msgp.AppendInt64(o, z.PageTables)
	// string "dirty"
	o = append(o, 0xa5, 0x64, 0x69, 0x72, 0x74, 0x79)
	o = msgp.AppendInt64(o, z.Dirty)
	// string "mapped"
	o = append(o, 0xa6, 0x6d, 0x61, 0x70, 0x70, 0x65, 0x64)
	o = msgp.AppendInt64(o, z.Mapped)
	// string "active"
	o = append(o, 0xa6, 0x61, 0x63, 0x74, 0x69, 0x76, 0x65)
	o = msgp.AppendInt64(o, z.Active)
	// string "total"
	o = append(o, 0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	o = msgp.AppendInt64(o, z.Total)
	// string "slab"
	o = append(o, 0xa4, 0x73, 0x6c, 0x61, 0x62)
	o = msgp.AppendInt64(o, z.Slab)
	// string "buffers"
	o = append(o, 0xa7, 0x62, 0x75, 0x66, 0x66, 0x65, 0x72, 0x73)
	o = msgp.AppendInt64(o, z.Buffers)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RdsOsMemory) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var pks uint32
	pks, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for pks > 0 {
		pks--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "writeback":
			z.Writeback, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "hugePagesFree":
			z.HugePagesFree, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "hugePagesRsvd":
			z.HugePagesRsvd, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "hugePagesSurp":
			z.HugePagesSurp, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "cached":
			z.Cached, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "hugePagesSize":
			z.HugePagesSize, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "free":
			z.Free, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "hugePagesTotal":
			z.HugePagesTotal, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "inactive":
			z.Inactive, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "pageTables":
			z.PageTables, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "dirty":
			z.Dirty, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "mapped":
			z.Mapped, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "active":
			z.Active, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "total":
			z.Total, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "slab":
			z.Slab, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "buffers":
			z.Buffers, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z *RdsOsMemory) Msgsize() (s int) {
	s = 3 + 10 + msgp.Int64Size + 14 + msgp.Int64Size + 14 + msgp.Int64Size + 14 + msgp.Int64Size + 7 + msgp.Int64Size + 14 + msgp.Int64Size + 5 + msgp.Int64Size + 15 + msgp.Int64Size + 9 + msgp.Int64Size + 11 + msgp.Int64Size + 6 + msgp.Int64Size + 7 + msgp.Int64Size + 7 + msgp.Int64Size + 6 + msgp.Int64Size + 5 + msgp.Int64Size + 8 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RdsOsNetworkInterface) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var jfb uint32
	jfb, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for jfb > 0 {
		jfb--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "interface":
			z.Interface, err = dc.ReadString()
			if err != nil {
				return
			}
		case "rx":
			z.Rx, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "tx":
			z.Tx, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z RdsOsNetworkInterface) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "interface"
	err = en.Append(0x83, 0xa9, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Interface)
	if err != nil {
		return
	}
	// write "rx"
	err = en.Append(0xa2, 0x72, 0x78)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Rx)
	if err != nil {
		return
	}
	// write "tx"
	err = en.Append(0xa2, 0x74, 0x78)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Tx)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z RdsOsNetworkInterface) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "interface"
	o = append(o, 0x83, 0xa9, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65)
	o = msgp.AppendString(o, z.Interface)
	// string "rx"
	o = append(o, 0xa2, 0x72, 0x78)
	o = msgp.AppendFloat64(o, z.Rx)
	// string "tx"
	o = append(o, 0xa2, 0x74, 0x78)
	o = msgp.AppendFloat64(o, z.Tx)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RdsOsNetworkInterface) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var cxo uint32
	cxo, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for cxo > 0 {
		cxo--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "interface":
			z.Interface, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "rx":
			z.Rx, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "tx":
			z.Tx, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z RdsOsNetworkInterface) Msgsize() (s int) {
	s = 1 + 10 + msgp.StringPrefixSize + len(z.Interface) + 3 + msgp.Float64Size + 3 + msgp.Float64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RdsOsProcess) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var eff uint32
	eff, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for eff > 0 {
		eff--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "vss":
			z.Vss, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "name":
			z.Name, err = dc.ReadString()
			if err != nil {
				return
			}
		case "tgid":
			z.Tgid, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "parentID":
			z.ParentID, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "memoryUsedPc":
			z.MemoryUsedPc, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "cpuUsedPc":
			z.CPUUsedPc, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "id":
			z.ID, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "rss":
			z.Rss, err = dc.ReadInt64()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RdsOsProcess) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 8
	// write "vss"
	err = en.Append(0x88, 0xa3, 0x76, 0x73, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Vss)
	if err != nil {
		return
	}
	// write "name"
	err = en.Append(0xa4, 0x6e, 0x61, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Name)
	if err != nil {
		return
	}
	// write "tgid"
	err = en.Append(0xa4, 0x74, 0x67, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Tgid)
	if err != nil {
		return
	}
	// write "parentID"
	err = en.Append(0xa8, 0x70, 0x61, 0x72, 0x65, 0x6e, 0x74, 0x49, 0x44)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.ParentID)
	if err != nil {
		return
	}
	// write "memoryUsedPc"
	err = en.Append(0xac, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x55, 0x73, 0x65, 0x64, 0x50, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.MemoryUsedPc)
	if err != nil {
		return
	}
	// write "cpuUsedPc"
	err = en.Append(0xa9, 0x63, 0x70, 0x75, 0x55, 0x73, 0x65, 0x64, 0x50, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.CPUUsedPc)
	if err != nil {
		return
	}
	// write "id"
	err = en.Append(0xa2, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.ID)
	if err != nil {
		return
	}
	// write "rss"
	err = en.Append(0xa3, 0x72, 0x73, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Rss)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RdsOsProcess) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 8
	// string "vss"
	o = append(o, 0x88, 0xa3, 0x76, 0x73, 0x73)
	o = msgp.AppendInt64(o, z.Vss)
	// string "name"
	o = append(o, 0xa4, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.Name)
	// string "tgid"
	o = append(o, 0xa4, 0x74, 0x67, 0x69, 0x64)
	o = msgp.AppendInt64(o, z.Tgid)
	// string "parentID"
	o = append(o, 0xa8, 0x70, 0x61, 0x72, 0x65, 0x6e, 0x74, 0x49, 0x44)
	o = msgp.AppendInt64(o, z.ParentID)
	// string "memoryUsedPc"
	o = append(o, 0xac, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x55, 0x73, 0x65, 0x64, 0x50, 0x63)
	o = msgp.AppendFloat32(o, z.MemoryUsedPc)
	// string "cpuUsedPc"
	o = append(o, 0xa9, 0x63, 0x70, 0x75, 0x55, 0x73, 0x65, 0x64, 0x50, 0x63)
	o = msgp.AppendFloat32(o, z.CPUUsedPc)
	// string "id"
	o = append(o, 0xa2, 0x69, 0x64)
	o = msgp.AppendInt64(o, z.ID)
	// string "rss"
	o = append(o, 0xa3, 0x72, 0x73, 0x73)
	o = msgp.AppendInt64(o, z.Rss)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RdsOsProcess) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var rsw uint32
	rsw, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for rsw > 0 {
		rsw--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "vss":
			z.Vss, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "name":
			z.Name, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "tgid":
			z.Tgid, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "parentID":
			z.ParentID, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "memoryUsedPc":
			z.MemoryUsedPc, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "cpuUsedPc":
			z.CPUUsedPc, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "id":
			z.ID, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "rss":
			z.Rss, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z *RdsOsProcess) Msgsize() (s int) {
	s = 1 + 4 + msgp.Int64Size + 5 + msgp.StringPrefixSize + len(z.Name) + 5 + msgp.Int64Size + 9 + msgp.Int64Size + 13 + msgp.Float32Size + 10 + msgp.Float32Size + 3 + msgp.Int64Size + 4 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RdsOsSnapshot) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var snv uint32
	snv, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for snv > 0 {
		snv--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "engine":
			z.Engine, err = dc.ReadString()
			if err != nil {
				return
			}
		case "instanceID":
			z.InstanceID, err = dc.ReadString()
			if err != nil {
				return
			}
		case "instanceResourceID":
			z.InstanceResourceID, err = dc.ReadString()
			if err != nil {
				return
			}
		case "timestamp":
			z.Timestamp, err = dc.ReadString()
			if err != nil {
				return
			}
		case "version":
			z.Version, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "uptime":
			z.Uptime, err = dc.ReadString()
			if err != nil {
				return
			}
		case "numVCPUs":
			z.NumVCPUs, err = dc.ReadInt32()
			if err != nil {
				return
			}
		case "cpuUtilization":
			err = z.CPUUtilization.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "loadAverageMinute":
			var kgt uint32
			kgt, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			for kgt > 0 {
				kgt--
				field, err = dc.ReadMapKeyPtr()
				if err != nil {
					return
				}
				switch msgp.UnsafeString(field) {
				case "fifteen":
					z.LoadAverageMinute.Fifteen, err = dc.ReadFloat32()
					if err != nil {
						return
					}
				case "five":
					z.LoadAverageMinute.Five, err = dc.ReadFloat32()
					if err != nil {
						return
					}
				case "one":
					z.LoadAverageMinute.One, err = dc.ReadFloat32()
					if err != nil {
						return
					}
				default:
					err = dc.Skip()
					if err != nil {
						return
					}
				}
			}
		case "memory":
			err = z.Memory.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "tasks":
			err = z.Tasks.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "swap":
			var ema uint32
			ema, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			for ema > 0 {
				ema--
				field, err = dc.ReadMapKeyPtr()
				if err != nil {
					return
				}
				switch msgp.UnsafeString(field) {
				case "cached":
					z.Swap.Cached, err = dc.ReadInt64()
					if err != nil {
						return
					}
				case "total":
					z.Swap.Total, err = dc.ReadInt64()
					if err != nil {
						return
					}
				case "free":
					z.Swap.Free, err = dc.ReadInt64()
					if err != nil {
						return
					}
				default:
					err = dc.Skip()
					if err != nil {
						return
					}
				}
			}
		case "network":
			var pez uint32
			pez, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Network) >= int(pez) {
				z.Network = z.Network[:pez]
			} else {
				z.Network = make([]RdsOsNetworkInterface, pez)
			}
			for xpk := range z.Network {
				var qke uint32
				qke, err = dc.ReadMapHeader()
				if err != nil {
					return
				}
				for qke > 0 {
					qke--
					field, err = dc.ReadMapKeyPtr()
					if err != nil {
						return
					}
					switch msgp.UnsafeString(field) {
					case "interface":
						z.Network[xpk].Interface, err = dc.ReadString()
						if err != nil {
							return
						}
					case "rx":
						z.Network[xpk].Rx, err = dc.ReadFloat64()
						if err != nil {
							return
						}
					case "tx":
						z.Network[xpk].Tx, err = dc.ReadFloat64()
						if err != nil {
							return
						}
					default:
						err = dc.Skip()
						if err != nil {
							return
						}
					}
				}
			}
		case "diskIO":
			var qyh uint32
			qyh, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.DiskIO) >= int(qyh) {
				z.DiskIO = z.DiskIO[:qyh]
			} else {
				z.DiskIO = make([]RdsOsDiskIO, qyh)
			}
			for dnj := range z.DiskIO {
				err = z.DiskIO[dnj].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "fileSys":
			var yzr uint32
			yzr, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.FileSystems) >= int(yzr) {
				z.FileSystems = z.FileSystems[:yzr]
			} else {
				z.FileSystems = make([]RdsOsFileSystem, yzr)
			}
			for obc := range z.FileSystems {
				err = z.FileSystems[obc].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RdsOsSnapshot) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 15
	// write "engine"
	err = en.Append(0x8f, 0xa6, 0x65, 0x6e, 0x67, 0x69, 0x6e, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Engine)
	if err != nil {
		return
	}
	// write "instanceID"
	err = en.Append(0xaa, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x49, 0x44)
	if err != nil {
		return err
	}
	err = en.WriteString(z.InstanceID)
	if err != nil {
		return
	}
	// write "instanceResourceID"
	err = en.Append(0xb2, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x49, 0x44)
	if err != nil {
		return err
	}
	err = en.WriteString(z.InstanceResourceID)
	if err != nil {
		return
	}
	// write "timestamp"
	err = en.Append(0xa9, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Timestamp)
	if err != nil {
		return
	}
	// write "version"
	err = en.Append(0xa7, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.Version)
	if err != nil {
		return
	}
	// write "uptime"
	err = en.Append(0xa6, 0x75, 0x70, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Uptime)
	if err != nil {
		return
	}
	// write "numVCPUs"
	err = en.Append(0xa8, 0x6e, 0x75, 0x6d, 0x56, 0x43, 0x50, 0x55, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt32(z.NumVCPUs)
	if err != nil {
		return
	}
	// write "cpuUtilization"
	err = en.Append(0xae, 0x63, 0x70, 0x75, 0x55, 0x74, 0x69, 0x6c, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = z.CPUUtilization.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "loadAverageMinute"
	// map header, size 3
	// write "fifteen"
	err = en.Append(0xb1, 0x6c, 0x6f, 0x61, 0x64, 0x41, 0x76, 0x65, 0x72, 0x61, 0x67, 0x65, 0x4d, 0x69, 0x6e, 0x75, 0x74, 0x65, 0x83, 0xa7, 0x66, 0x69, 0x66, 0x74, 0x65, 0x65, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.LoadAverageMinute.Fifteen)
	if err != nil {
		return
	}
	// write "five"
	err = en.Append(0xa4, 0x66, 0x69, 0x76, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.LoadAverageMinute.Five)
	if err != nil {
		return
	}
	// write "one"
	err = en.Append(0xa3, 0x6f, 0x6e, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat32(z.LoadAverageMinute.One)
	if err != nil {
		return
	}
	// write "memory"
	err = en.Append(0xa6, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79)
	if err != nil {
		return err
	}
	err = z.Memory.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "tasks"
	err = en.Append(0xa5, 0x74, 0x61, 0x73, 0x6b, 0x73)
	if err != nil {
		return err
	}
	err = z.Tasks.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "swap"
	// map header, size 3
	// write "cached"
	err = en.Append(0xa4, 0x73, 0x77, 0x61, 0x70, 0x83, 0xa6, 0x63, 0x61, 0x63, 0x68, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Swap.Cached)
	if err != nil {
		return
	}
	// write "total"
	err = en.Append(0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Swap.Total)
	if err != nil {
		return
	}
	// write "free"
	err = en.Append(0xa4, 0x66, 0x72, 0x65, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Swap.Free)
	if err != nil {
		return
	}
	// write "network"
	err = en.Append(0xa7, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Network)))
	if err != nil {
		return
	}
	for xpk := range z.Network {
		// map header, size 3
		// write "interface"
		err = en.Append(0x83, 0xa9, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65)
		if err != nil {
			return err
		}
		err = en.WriteString(z.Network[xpk].Interface)
		if err != nil {
			return
		}
		// write "rx"
		err = en.Append(0xa2, 0x72, 0x78)
		if err != nil {
			return err
		}
		err = en.WriteFloat64(z.Network[xpk].Rx)
		if err != nil {
			return
		}
		// write "tx"
		err = en.Append(0xa2, 0x74, 0x78)
		if err != nil {
			return err
		}
		err = en.WriteFloat64(z.Network[xpk].Tx)
		if err != nil {
			return
		}
	}
	// write "diskIO"
	err = en.Append(0xa6, 0x64, 0x69, 0x73, 0x6b, 0x49, 0x4f)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.DiskIO)))
	if err != nil {
		return
	}
	for dnj := range z.DiskIO {
		err = z.DiskIO[dnj].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "fileSys"
	err = en.Append(0xa7, 0x66, 0x69, 0x6c, 0x65, 0x53, 0x79, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.FileSystems)))
	if err != nil {
		return
	}
	for obc := range z.FileSystems {
		err = z.FileSystems[obc].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RdsOsSnapshot) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 15
	// string "engine"
	o = append(o, 0x8f, 0xa6, 0x65, 0x6e, 0x67, 0x69, 0x6e, 0x65)
	o = msgp.AppendString(o, z.Engine)
	// string "instanceID"
	o = append(o, 0xaa, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x49, 0x44)
	o = msgp.AppendString(o, z.InstanceID)
	// string "instanceResourceID"
	o = append(o, 0xb2, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x49, 0x44)
	o = msgp.AppendString(o, z.InstanceResourceID)
	// string "timestamp"
	o = append(o, 0xa9, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70)
	o = msgp.AppendString(o, z.Timestamp)
	// string "version"
	o = append(o, 0xa7, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	o = msgp.AppendFloat32(o, z.Version)
	// string "uptime"
	o = append(o, 0xa6, 0x75, 0x70, 0x74, 0x69, 0x6d, 0x65)
	o = msgp.AppendString(o, z.Uptime)
	// string "numVCPUs"
	o = append(o, 0xa8, 0x6e, 0x75, 0x6d, 0x56, 0x43, 0x50, 0x55, 0x73)
	o = msgp.AppendInt32(o, z.NumVCPUs)
	// string "cpuUtilization"
	o = append(o, 0xae, 0x63, 0x70, 0x75, 0x55, 0x74, 0x69, 0x6c, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	o, err = z.CPUUtilization.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "loadAverageMinute"
	// map header, size 3
	// string "fifteen"
	o = append(o, 0xb1, 0x6c, 0x6f, 0x61, 0x64, 0x41, 0x76, 0x65, 0x72, 0x61, 0x67, 0x65, 0x4d, 0x69, 0x6e, 0x75, 0x74, 0x65, 0x83, 0xa7, 0x66, 0x69, 0x66, 0x74, 0x65, 0x65, 0x6e)
	o = msgp.AppendFloat32(o, z.LoadAverageMinute.Fifteen)
	// string "five"
	o = append(o, 0xa4, 0x66, 0x69, 0x76, 0x65)
	o = msgp.AppendFloat32(o, z.LoadAverageMinute.Five)
	// string "one"
	o = append(o, 0xa3, 0x6f, 0x6e, 0x65)
	o = msgp.AppendFloat32(o, z.LoadAverageMinute.One)
	// string "memory"
	o = append(o, 0xa6, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79)
	o, err = z.Memory.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "tasks"
	o = append(o, 0xa5, 0x74, 0x61, 0x73, 0x6b, 0x73)
	o, err = z.Tasks.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "swap"
	// map header, size 3
	// string "cached"
	o = append(o, 0xa4, 0x73, 0x77, 0x61, 0x70, 0x83, 0xa6, 0x63, 0x61, 0x63, 0x68, 0x65, 0x64)
	o = msgp.AppendInt64(o, z.Swap.Cached)
	// string "total"
	o = append(o, 0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	o = msgp.AppendInt64(o, z.Swap.Total)
	// string "free"
	o = append(o, 0xa4, 0x66, 0x72, 0x65, 0x65)
	o = msgp.AppendInt64(o, z.Swap.Free)
	// string "network"
	o = append(o, 0xa7, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Network)))
	for xpk := range z.Network {
		// map header, size 3
		// string "interface"
		o = append(o, 0x83, 0xa9, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65)
		o = msgp.AppendString(o, z.Network[xpk].Interface)
		// string "rx"
		o = append(o, 0xa2, 0x72, 0x78)
		o = msgp.AppendFloat64(o, z.Network[xpk].Rx)
		// string "tx"
		o = append(o, 0xa2, 0x74, 0x78)
		o = msgp.AppendFloat64(o, z.Network[xpk].Tx)
	}
	// string "diskIO"
	o = append(o, 0xa6, 0x64, 0x69, 0x73, 0x6b, 0x49, 0x4f)
	o = msgp.AppendArrayHeader(o, uint32(len(z.DiskIO)))
	for dnj := range z.DiskIO {
		o, err = z.DiskIO[dnj].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "fileSys"
	o = append(o, 0xa7, 0x66, 0x69, 0x6c, 0x65, 0x53, 0x79, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.FileSystems)))
	for obc := range z.FileSystems {
		o, err = z.FileSystems[obc].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RdsOsSnapshot) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var ywj uint32
	ywj, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for ywj > 0 {
		ywj--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "engine":
			z.Engine, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "instanceID":
			z.InstanceID, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "instanceResourceID":
			z.InstanceResourceID, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "timestamp":
			z.Timestamp, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "version":
			z.Version, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "uptime":
			z.Uptime, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "numVCPUs":
			z.NumVCPUs, bts, err = msgp.ReadInt32Bytes(bts)
			if err != nil {
				return
			}
		case "cpuUtilization":
			bts, err = z.CPUUtilization.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "loadAverageMinute":
			var jpj uint32
			jpj, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			for jpj > 0 {
				jpj--
				field, bts, err = msgp.ReadMapKeyZC(bts)
				if err != nil {
					return
				}
				switch msgp.UnsafeString(field) {
				case "fifteen":
					z.LoadAverageMinute.Fifteen, bts, err = msgp.ReadFloat32Bytes(bts)
					if err != nil {
						return
					}
				case "five":
					z.LoadAverageMinute.Five, bts, err = msgp.ReadFloat32Bytes(bts)
					if err != nil {
						return
					}
				case "one":
					z.LoadAverageMinute.One, bts, err = msgp.ReadFloat32Bytes(bts)
					if err != nil {
						return
					}
				default:
					bts, err = msgp.Skip(bts)
					if err != nil {
						return
					}
				}
			}
		case "memory":
			bts, err = z.Memory.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "tasks":
			bts, err = z.Tasks.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "swap":
			var zpf uint32
			zpf, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			for zpf > 0 {
				zpf--
				field, bts, err = msgp.ReadMapKeyZC(bts)
				if err != nil {
					return
				}
				switch msgp.UnsafeString(field) {
				case "cached":
					z.Swap.Cached, bts, err = msgp.ReadInt64Bytes(bts)
					if err != nil {
						return
					}
				case "total":
					z.Swap.Total, bts, err = msgp.ReadInt64Bytes(bts)
					if err != nil {
						return
					}
				case "free":
					z.Swap.Free, bts, err = msgp.ReadInt64Bytes(bts)
					if err != nil {
						return
					}
				default:
					bts, err = msgp.Skip(bts)
					if err != nil {
						return
					}
				}
			}
		case "network":
			var rfe uint32
			rfe, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Network) >= int(rfe) {
				z.Network = z.Network[:rfe]
			} else {
				z.Network = make([]RdsOsNetworkInterface, rfe)
			}
			for xpk := range z.Network {
				var gmo uint32
				gmo, bts, err = msgp.ReadMapHeaderBytes(bts)
				if err != nil {
					return
				}
				for gmo > 0 {
					gmo--
					field, bts, err = msgp.ReadMapKeyZC(bts)
					if err != nil {
						return
					}
					switch msgp.UnsafeString(field) {
					case "interface":
						z.Network[xpk].Interface, bts, err = msgp.ReadStringBytes(bts)
						if err != nil {
							return
						}
					case "rx":
						z.Network[xpk].Rx, bts, err = msgp.ReadFloat64Bytes(bts)
						if err != nil {
							return
						}
					case "tx":
						z.Network[xpk].Tx, bts, err = msgp.ReadFloat64Bytes(bts)
						if err != nil {
							return
						}
					default:
						bts, err = msgp.Skip(bts)
						if err != nil {
							return
						}
					}
				}
			}
		case "diskIO":
			var taf uint32
			taf, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.DiskIO) >= int(taf) {
				z.DiskIO = z.DiskIO[:taf]
			} else {
				z.DiskIO = make([]RdsOsDiskIO, taf)
			}
			for dnj := range z.DiskIO {
				bts, err = z.DiskIO[dnj].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "fileSys":
			var eth uint32
			eth, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.FileSystems) >= int(eth) {
				z.FileSystems = z.FileSystems[:eth]
			} else {
				z.FileSystems = make([]RdsOsFileSystem, eth)
			}
			for obc := range z.FileSystems {
				bts, err = z.FileSystems[obc].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z *RdsOsSnapshot) Msgsize() (s int) {
	s = 1 + 7 + msgp.StringPrefixSize + len(z.Engine) + 11 + msgp.StringPrefixSize + len(z.InstanceID) + 19 + msgp.StringPrefixSize + len(z.InstanceResourceID) + 10 + msgp.StringPrefixSize + len(z.Timestamp) + 8 + msgp.Float32Size + 7 + msgp.StringPrefixSize + len(z.Uptime) + 9 + msgp.Int32Size + 15 + z.CPUUtilization.Msgsize() + 18 + 1 + 8 + msgp.Float32Size + 5 + msgp.Float32Size + 4 + msgp.Float32Size + 7 + z.Memory.Msgsize() + 6 + z.Tasks.Msgsize() + 5 + 1 + 7 + msgp.Int64Size + 6 + msgp.Int64Size + 5 + msgp.Int64Size + 8 + msgp.ArrayHeaderSize
	for xpk := range z.Network {
		s += 1 + 10 + msgp.StringPrefixSize + len(z.Network[xpk].Interface) + 3 + msgp.Float64Size + 3 + msgp.Float64Size
	}
	s += 7 + msgp.ArrayHeaderSize
	for dnj := range z.DiskIO {
		s += z.DiskIO[dnj].Msgsize()
	}
	s += 8 + msgp.ArrayHeaderSize
	for obc := range z.FileSystems {
		s += z.FileSystems[obc].Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RdsOsSwap) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var sbz uint32
	sbz, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for sbz > 0 {
		sbz--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "cached":
			z.Cached, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "total":
			z.Total, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "free":
			z.Free, err = dc.ReadInt64()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z RdsOsSwap) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "cached"
	err = en.Append(0x83, 0xa6, 0x63, 0x61, 0x63, 0x68, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Cached)
	if err != nil {
		return
	}
	// write "total"
	err = en.Append(0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Total)
	if err != nil {
		return
	}
	// write "free"
	err = en.Append(0xa4, 0x66, 0x72, 0x65, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Free)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z RdsOsSwap) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "cached"
	o = append(o, 0x83, 0xa6, 0x63, 0x61, 0x63, 0x68, 0x65, 0x64)
	o = msgp.AppendInt64(o, z.Cached)
	// string "total"
	o = append(o, 0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	o = msgp.AppendInt64(o, z.Total)
	// string "free"
	o = append(o, 0xa4, 0x66, 0x72, 0x65, 0x65)
	o = msgp.AppendInt64(o, z.Free)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RdsOsSwap) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var rjx uint32
	rjx, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for rjx > 0 {
		rjx--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "cached":
			z.Cached, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "total":
			z.Total, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "free":
			z.Free, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z RdsOsSwap) Msgsize() (s int) {
	s = 1 + 7 + msgp.Int64Size + 6 + msgp.Int64Size + 5 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RdsOsTasks) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var awn uint32
	awn, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for awn > 0 {
		awn--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "sleeping":
			z.Sleeping, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "zombie":
			z.Zombie, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "running":
			z.Running, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "stopped":
			z.Stopped, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "total":
			z.Total, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "blocked":
			z.Blocked, err = dc.ReadInt64()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RdsOsTasks) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "sleeping"
	err = en.Append(0x86, 0xa8, 0x73, 0x6c, 0x65, 0x65, 0x70, 0x69, 0x6e, 0x67)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Sleeping)
	if err != nil {
		return
	}
	// write "zombie"
	err = en.Append(0xa6, 0x7a, 0x6f, 0x6d, 0x62, 0x69, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Zombie)
	if err != nil {
		return
	}
	// write "running"
	err = en.Append(0xa7, 0x72, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Running)
	if err != nil {
		return
	}
	// write "stopped"
	err = en.Append(0xa7, 0x73, 0x74, 0x6f, 0x70, 0x70, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Stopped)
	if err != nil {
		return
	}
	// write "total"
	err = en.Append(0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Total)
	if err != nil {
		return
	}
	// write "blocked"
	err = en.Append(0xa7, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Blocked)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RdsOsTasks) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "sleeping"
	o = append(o, 0x86, 0xa8, 0x73, 0x6c, 0x65, 0x65, 0x70, 0x69, 0x6e, 0x67)
	o = msgp.AppendInt64(o, z.Sleeping)
	// string "zombie"
	o = append(o, 0xa6, 0x7a, 0x6f, 0x6d, 0x62, 0x69, 0x65)
	o = msgp.AppendInt64(o, z.Zombie)
	// string "running"
	o = append(o, 0xa7, 0x72, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67)
	o = msgp.AppendInt64(o, z.Running)
	// string "stopped"
	o = append(o, 0xa7, 0x73, 0x74, 0x6f, 0x70, 0x70, 0x65, 0x64)
	o = msgp.AppendInt64(o, z.Stopped)
	// string "total"
	o = append(o, 0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	o = msgp.AppendInt64(o, z.Total)
	// string "blocked"
	o = append(o, 0xa7, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64)
	o = msgp.AppendInt64(o, z.Blocked)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RdsOsTasks) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var wel uint32
	wel, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for wel > 0 {
		wel--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "sleeping":
			z.Sleeping, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "zombie":
			z.Zombie, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "running":
			z.Running, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "stopped":
			z.Stopped, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "total":
			z.Total, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "blocked":
			z.Blocked, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z *RdsOsTasks) Msgsize() (s int) {
	s = 1 + 9 + msgp.Int64Size + 7 + msgp.Int64Size + 8 + msgp.Int64Size + 8 + msgp.Int64Size + 6 + msgp.Int64Size + 8 + msgp.Int64Size
	return
}
