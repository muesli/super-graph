package serv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dosco/super-graph/migrate"
	"github.com/spf13/cobra"
)

var sampleMigration = `-- This is a sample migration.

create table users(
  id serial primary key,
  fullname varchar not null,
  email varchar not null
);

---- create above / drop below ----

drop table users;
`

var newMigrationText = `-- Write your migrate up statements here

---- create above / drop below ----

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
`

func cmdNewMigration(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmd.Help()
		os.Exit(1)
	}

	name := args[0]

	m, err := migrate.FindMigrations(conf.MigrationsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading migrations:\n  %v\n", err)
		os.Exit(1)
	}

	mname := fmt.Sprintf("%03d_%s.sql", len(m)+100, name)

	// Write new migration
	mpath := filepath.Join(conf.MigrationsPath, mname)
	mfile, err := os.OpenFile(mpath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer mfile.Close()

	_, err = mfile.WriteString(newMigrationText)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger.Info().Msgf("created migration '%s'\n", mpath)
}

func cmdMigrate(cmd *cobra.Command, args []string) {
	conn, err := initDB(conf)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer conn.Close(context.Background())

	m, err := migrate.NewMigrator(conn, "schema_version")
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initializing migrator")
	}
	//m.Data = config.Data

	err = m.LoadMigrations(conf.MigrationsPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load migrations")
	}

	if len(m.Migrations) == 0 {
		logger.Fatal().Msg("No migrations found")
	}

	m.OnStart = func(sequence int32, name, direction, sql string) {
		logger.Info().Msgf("%s executing %s %s\n%s\n\n",
			time.Now().Format("2006-01-02 15:04:05"), name, direction, sql)
	}

	var currentVersion int32
	currentVersion, err = m.GetCurrentVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get current version:\n  %v\n", err)
		os.Exit(1)
	}

	dest := args[0]
	mustParseDestination := func(d string) int32 {
		var n int64
		n, err = strconv.ParseInt(d, 10, 32)
		if err != nil {
			logger.Fatal().Err(err).Msg("invalid destination")
		}
		return int32(n)
	}

	if dest == "last" {
		err = m.Migrate()

	} else if len(dest) >= 3 && dest[0:2] == "-+" {
		err = m.MigrateTo(currentVersion - mustParseDestination(dest[2:]))
		if err == nil {
			err = m.MigrateTo(currentVersion)
		}

	} else if len(dest) >= 2 && dest[0] == '-' {
		err = m.MigrateTo(currentVersion - mustParseDestination(dest[1:]))

	} else if len(dest) >= 2 && dest[0] == '+' {
		err = m.MigrateTo(currentVersion + mustParseDestination(dest[1:]))

	} else {
		//err = make(type, 0).MigrateTo(mustParseDestination(dest))
	}

	if err != nil {
		logger.Info().Err(err).Send()

		// logger.Info().Err(err).Send()

		// if err, ok := err.(m.MigrationPgError); ok {
		// 	if err.Detail != "" {
		// 		logger.Info().Err(err).Msg(err.Detail)
		// 	}

		// 	if err.Position != 0 {
		// 		ele, err := ExtractErrorLine(err.Sql, int(err.Position))
		// 		if err != nil {
		// 			logger.Fatal().Err(err).Send()
		// 		}

		// 		prefix := fmt.Sprintf()
		// 		logger.Info().Msgf("line %d, %s%s", ele.LineNum, prefix, ele.Text)
		// 	}
		// }
		// os.Exit(1)
	}
	logger.Info().Msg("migration done")

}

func cmdStatus(cmd *cobra.Command, args []string) {
	conn, err := initDB(conf)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer conn.Close(context.Background())

	m, err := migrate.NewMigrator(conn, "schema_version")
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize migrator")
	}
	//m.Data = config.Data

	err = m.LoadMigrations(conf.MigrationsPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load migrations")
	}

	if len(m.Migrations) == 0 {
		logger.Fatal().Msg("no migrations found")
	}

	mver, err := m.GetCurrentVersion()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to retrieve migration")
	}

	var status string
	behindCount := len(m.Migrations) - int(mver)
	if behindCount == 0 {
		status = "up to date"
	} else {
		status = "migration(s) pending"
	}

	fmt.Println("status:  ", status)
	fmt.Println("version:  %d of %d\n", mver, len(m.Migrations))
	fmt.Println("host:    ", conf.DB.Host)
	fmt.Println("database:", conf.DB.DBName)
}

type ErrorLineExtract struct {
	LineNum   int    // Line number starting with 1
	ColumnNum int    // Column number starting with 1
	Text      string // Text of the line without a new line character.
}

// ExtractErrorLine takes source and character position extracts the line
// number, column number, and the line of text.
//
// The first character is position 1.
func ExtractErrorLine(source string, position int) (ErrorLineExtract, error) {
	ele := ErrorLineExtract{LineNum: 1}

	if position > len(source) {
		return ele, fmt.Errorf("position (%d) is greater than source length (%d)", position, len(source))
	}

	lines := strings.SplitAfter(source, "\n")
	for _, ele.Text = range lines {
		if position-len(ele.Text) < 1 {
			ele.ColumnNum = position
			break
		}

		ele.LineNum += 1
		position -= len(ele.Text)
	}

	ele.Text = strings.TrimSuffix(ele.Text, "\n")

	return ele, nil
}