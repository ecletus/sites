package models

import "github.com/moisespsena-go/aorm"

type SiteConfig struct {
	ID aorm.StrId
	aorm.Audited
	Value string `sql:"text"`
	_ interface{} `aorm:"table_name:site_config"`
}
