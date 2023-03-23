package oiajudge

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/carlosmiguelsoto/oiajudge/pkg/bridge"
	"github.com/carlosmiguelsoto/oiajudge/pkg/store"
	"github.com/carlosmiguelsoto/oiajudge/pkg/utils"
)

type Config struct {
	OiaDbConnectionString string
	OiaServerPort         int64
}

//go:embed migrations
var migrations embed.FS

func RunServer(ctx context.Context, bridge bridge.Bridge) error {
	port_string := os.Getenv("OIAJ_SERVER_PORT")
	port, err := strconv.ParseInt(port_string, 10, 64)
	if err != nil {
		return err
	}
	config := Config{
		OiaDbConnectionString: os.Getenv("OIAJ_DB_CONNECTION_STRING"),
		OiaServerPort:         port,
	}

	sql, err := utils.ExtractEmbeddedFsIntoFileMap(migrations, "migrations")
	if err != nil {
		return err
	}
	client, err := store.MakeClientWithInitScript(ctx, config.OiaDbConnectionString, sql, "oiajudge")
	if err != nil {
		return err
	}

	server := &Server{
		Db:     client,
		Bridge: bridge,
		Config: config,
	}

	bridge.HandleEvents(context.Background(), server.HandleEvents)

	handler := server.MakeServer()
	url := fmt.Sprintf(":%d", config.OiaServerPort)
	err = http.ListenAndServe(url, handler)
	return err
}
