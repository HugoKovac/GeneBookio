package database

import (
	"context"
	"fmt"
	"hkorpo/book/pkg/ent"
	"hkorpo/book/pkg/errorwrapper"

	_ "github.com/go-sql-driver/mysql"
)

func Init(cfg *ConfigDB) (*ent.Client, error) {
	ctx := context.Background()
	dbClient, err := ent.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=True",
		cfg.USER,
		cfg.PASSWORD,
		cfg.HOST,
		cfg.DATABASE,
	))
	if err != nil {
		return nil, errorwrapper.Wrap(fmt.Errorf("failed opening connection to mysql: %v", err))
	}

	if err := dbClient.Schema.Create(ctx); err != nil {
		return nil, errorwrapper.Wrap(fmt.Errorf("database migration failed: %v", err))
	}

	return dbClient, nil
}
