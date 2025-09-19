package store

import "time"

// GORM models
type Job struct {
	ID         uint   `gorm:"primaryKey"`
	Name       string `gorm:"type:varchar(64);unique;not null;comment:名称"`
	CronExpr   string `gorm:"type:varchar(256);not null;comment:定时任务表达式"`
	Target     string `gorm:"type:varchar(256);not null;comment:执行目标"` // http(s)... or cmd://...
	Parallel   int    `gorm:"type:int(256);not null;default:1"`
	RetryCount int    `gorm:"not null;default:0"`
	TimeoutSec int    `gorm:"not null;default:5"`
	Enabled    bool   `gorm:"not null;default:true"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ExecRecord struct {
	ID        uint      `gorm:"primaryKey"`
	JobID     uint      `gorm:"index;not null"`
	StartTime time.Time `gorm:"not null"`
	EndTime   *time.Time
	Success   bool `gorm:"type:tinyint(1);not null"`
	Attempt   int
	Output    string `gorm:"type:text"`
	CreatedAt time.Time
}
