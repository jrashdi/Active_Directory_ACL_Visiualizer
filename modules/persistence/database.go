package persistence

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lkarlslund/adalanche/modules/cli"
	"github.com/lkarlslund/adalanche/modules/ui"
	"github.com/spf13/cobra"
	"github.com/ugorji/go/codec"
	"go.etcd.io/bbolt"
)

var (
	datastore *bbolt.DB
	mh        codec.JsonHandle
)

var (
	persistenceCmd = &cobra.Command{
		Use:   "persistence",
		Short: "Maintenance tools for the persistence database",
	}
	dumpCmd = &cobra.Command{
		Use:   "dump",
		Short: "Dumps the persistence database in JSON",
	}
	output = dumpCmd.Flags().String("output", "persistence-dump.json", "Output file for dump")

	restoreCmd = &cobra.Command{
		Use:   "restore",
		Short: "Restores the persistence database from JSON",
	}
	input = restoreCmd.Flags().String("input", "persistence-dump.json", "Input file to restore")
)

func init() {
	cli.Root.AddCommand(persistenceCmd)
	persistenceCmd.AddCommand(dumpCmd)
	dumpCmd.RunE = dump
	persistenceCmd.AddCommand(restoreCmd)
	restoreCmd.RunE = restore
}

func getDB() (*bbolt.DB, error) {
	if datastore != nil {
		return datastore, nil
	}

	var err error
	datastore, err = bbolt.Open(filepath.Join(*cli.Datapath, "persistence.bbolt"), 0666, nil)
	return datastore, err

	// pogreb.SetLogger(ui.New(zerolog.ConsoleWriter{
	// 	Out:        colorable.NewColorableStdout(),
	// 	TimeFormat: "15:04:05.000",
	// }, "", 0))
}

type Identifiable interface {
	ID() string
}

type storage[i Identifiable] struct {
	db         *bbolt.DB
	bucketname []byte
	cache      map[string]i
}

func GetStorage[i Identifiable](bucketname string, cached bool) storage[i] {
	db, err := getDB()
	if err != nil {
		panic(err) // FIXME
	}
	s := storage[i]{
		db:         db,
		bucketname: []byte(bucketname),
		// cache:      make(map[string]i),
	}
	if cached {
		s.cache = make(map[string]i)
	}
	return s
}

func (s storage[p]) Get(id string) (*p, bool) {
	var result p
	if s.cache != nil {
		if rv, found := s.cache[string(id)]; found {
			return &rv, true
		}
	}

	var data []byte
	if s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(s.bucketname))
		if b == nil {
			return nil
		}
		data = b.Get([]byte(id))
		return nil
	}); data != nil {
		dec := codec.NewDecoderBytes(data, &mh)
		err := dec.Decode(&result)
		if err != nil {
			return nil, false
		}
		if s.cache != nil {
			s.cache[string(id)] = result
		}
		return &result, true
	}
	return nil, false
}

func (s storage[p]) Put(saveme p) {
	var output []byte
	enc := codec.NewEncoderBytes(&output, &mh)
	err := enc.Encode(saveme)
	if err != nil {
		ui.Fatal().Msgf("failed to marshal %v", saveme)
	}
	id := saveme.ID()

	err = s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(s.bucketname))
		if err != nil {
			return err
		}
		if b == nil {
			return nil
		}
		b.Put([]byte(id), output)
		return nil
	})
	if err != nil {
		ui.Fatal().Msgf("failed to put %v", saveme)
	}
	if s.cache != nil {
		s.cache[string(id)] = saveme
	}
}

func (s storage[p]) Delete(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(s.bucketname))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(id))
	})
}

func (s storage[p]) List() ([]p, error) {
	var result []p
	return result, s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(s.bucketname))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var data p
			dec := codec.NewDecoderBytes(v, &mh)
			err := dec.Decode(&data)
			if err != nil {
				return err
			}
			result = append(result, data)
			return nil
		})
	})
}

func dump(cmd *cobra.Command, args []string) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("Could not open database: %v", err)
	}

	// Open output file for writing
	jsonfile, err := os.Create(*output)
	if err != nil {
		return fmt.Errorf("Could not open output file: %v", err)
	}

	fmt.Fprintln(jsonfile, "[")

	// Iterate over all buckets, and dump all the data
	db.View(func(tx *bbolt.Tx) error {
		firstbucket := true
		tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
			if !firstbucket {
				fmt.Fprint(jsonfile, ",\n")
			} else {
				fmt.Fprint(jsonfile, "\n")
				firstbucket = false
			}
			fmt.Fprintf(jsonfile, "  {\n \"%v\":\n    {\n", string(name))
			c := b.Cursor()
			firstrecord := true
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if !firstrecord {
					fmt.Fprintf(jsonfile, ",\n")
				} else {
					firstrecord = false
					fmt.Fprintf(jsonfile, "\n")
				}
				fmt.Fprintf(jsonfile, "      \"%v\": %v", string(k), string(v))
			}
			fmt.Fprintf(jsonfile, "  }")
			return nil
		})
		return nil
	})

	fmt.Fprintln(jsonfile, "]")
	jsonfile.Close()

	return nil
}

func restore(cmd *cobra.Command, args []string) error {
	return nil
}