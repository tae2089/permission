package audit

import (
	"context"

	"github.com/tae2089/trace/v3"
	"gorm.io/gorm"
)

func Migrate(ctx context.Context, db *gorm.DB) error {
	if err := db.WithContext(ctx).AutoMigrate(&record{}); err != nil {
		return trace.Wrap(err, "migrate audit events")
	}
	return nil
}
