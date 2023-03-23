package cmsbridge

import (
	"embed"

	"github.com/carlosmiguelsoto/oiajudge/pkg/bridge"
	"github.com/carlosmiguelsoto/oiajudge/pkg/store"
	"github.com/carlosmiguelsoto/oiajudge/pkg/utils"

	"context"
	"log"
	"os"
)

type Config struct {
	StaticAssetDirectory string
	DbConnectionString   string
	CmsBridgeAddress     string
	OiaSubmitterAddress  string
	CmsContestId         bridge.Id
}

type CmsBridge struct {
	Config Config
	Db     store.DBClient
}

//go:embed migrations/*
var migrations embed.FS

func CreateCmsBridge() (bridge.Bridge, error) {
	ctx := context.Background()
	config := Config{
		DbConnectionString: os.Getenv("OIAJ_DB_CONNECTION_STRING"),
		CmsContestId:       1,
		CmsBridgeAddress:   os.Getenv("OIAJ_CMS_BRIDGE_ADDRESS"),
	}

	sql, err := utils.ExtractEmbeddedFsIntoFileMap(migrations, "migrations")
	if err != nil {
		return nil, err
	}

	db, err := store.MakeClientWithInitScript(ctx, config.DbConnectionString, sql, "cmsbridge")
	if err != nil {
		log.Fatal(err)
	}
	return &CmsBridge{
		Config: config,
		Db:     db,
	}, nil
}
