package models

// SystemSetting stores system configuration as key-value pairs.
// Each unique Key has one row. The Value stores JSON-encoded data.
// This is a general-purpose config persistence table.
type SystemSetting struct {
	ID    uint   `gorm:"primarykey" json:"id"`
	Key   string `gorm:"type:varchar(100);uniqueIndex;not null" json:"key"`
	Value string `gorm:"type:text;not null" json:"value"` // JSON-encoded
}

// TableName specifies the table name for SystemSetting
func (SystemSetting) TableName() string {
	return "system_settings"
}
