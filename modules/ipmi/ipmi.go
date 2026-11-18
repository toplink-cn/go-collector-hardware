package ipmi

import (
	"collector/bin"
	"collector/modules/collection"
	"context"
	"errors"
	"fmt"
	"os"
)

type Sensor struct {
	Key   string
	Value string
}

func Collect(ctx context.Context) (collection.Result[[]Sensor], error) {
	if err := platformAvailability(); err != nil {
		return collection.Result[[]Sensor]{Data: []Sensor{}, Quality: map[string]collection.Quality{"ipmi": collection.FromError(err, false)}}, err
	}
	output, err := bin.RunCommandContext(ctx, ipmiExecutable(), "sensor")
	values := []Sensor{}
	if err == nil {
		values = parseSensors(string(output))
		if len(values) == 0 {
			err = collection.ErrUnsupported
		}
	}
	if errors.Is(err, os.ErrNotExist) {
		err = fmt.Errorf("IPMI tool: %w", collection.ErrUnsupported)
	}
	return collection.Result[[]Sensor]{Data: values, Quality: map[string]collection.Quality{"ipmi": collection.FromError(err, len(values) > 0)}}, err
}
func GetInfo() []Sensor { result, _ := Collect(context.Background()); return result.Data }
