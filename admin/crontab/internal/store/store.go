package store

import (
	"context"

	"gorm.io/gorm"
)

type Store interface {
	ListJobs(ctx context.Context) ([]Job, error)
	FindJob(ctx context.Context, id uint) (*Job, error)
	CreateJob(ctx context.Context, j *Job) error
	InsertExec(ctx context.Context, r *ExecRecord) error
	AutoMigrate(db *gorm.DB) error
}

type gormStore struct {
	db *gorm.DB
}

func NewGormStore(db *gorm.DB) Store {
	return &gormStore{db: db}
}

func (s *gormStore) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Job{}, &ExecRecord{})
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Job{}, &ExecRecord{})
}

// 列表
func (s *gormStore) ListJobs(ctx context.Context) ([]Job, error) {
	var jobs []Job
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// 查找
func (s *gormStore) FindJob(ctx context.Context, id uint) (*Job, error) {
	var j Job
	if err := s.db.WithContext(ctx).First(&j, id).Error; err != nil {
		return nil, err
	}
	return &j, nil
}

// 创建
func (s *gormStore) CreateJob(ctx context.Context, j *Job) error {
	return s.db.WithContext(ctx).Create(j).Error
}

// 添加
func (s *gormStore) InsertExec(ctx context.Context, r *ExecRecord) error {
	return s.db.WithContext(ctx).Create(r).Error
}
