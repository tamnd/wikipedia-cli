package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/wikipedia-cli/wiki"
)

func newGeoSearchCmd(app *App) *cobra.Command {
	var radius int
	cmd := &cobra.Command{
		Use:     "geosearch <lat,lon | lat lon>",
		Aliases: []string{"geo"},
		Short:   "Find articles near a coordinate",
		Long: `Find articles within a radius of a latitude/longitude. Accept "lat,lon" as
one argument or as two. Radius is in metres (max 10000).

Examples:
  wiki geosearch 48.8584,2.2945 --radius 2000
  wiki geosearch 51.5007 -0.1246 -n 20 -o jsonl`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := app.Client()
			if err != nil {
				return err
			}
			lat, lon, err := parseLatLon(args)
			if err != nil {
				return usageErr(err.Error())
			}
			results, err := c.GeoSearch(cmd.Context(), lat, lon, radius, app.Limit)
			if err != nil {
				return wrapErr(err)
			}
			return emitGeo(app, results)
		},
	}
	cmd.Flags().IntVar(&radius, "radius", 1000, "search radius in metres (max 10000)")
	return cmd
}

func newNearbyCmd(app *App) *cobra.Command {
	var radius int
	cmd := &cobra.Command{
		Use:   "nearby <title>",
		Short: "Find articles near another article",
		Long: `Find articles geographically near a given article.

Examples:
  wiki nearby "Eiffel Tower" --radius 1000
  wiki nearby "Statue of Liberty" -n 15`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, title, err := app.resolveTarget(args[0])
			if err != nil {
				return err
			}
			results, err := c.GeoNear(cmd.Context(), title, radius, app.Limit)
			if err != nil {
				return wrapErr(err)
			}
			return emitGeo(app, results)
		},
	}
	cmd.Flags().IntVar(&radius, "radius", 1000, "search radius in metres (max 10000)")
	return cmd
}

func emitGeo(app *App, results []wiki.GeoResult) error {
	if len(results) == 0 {
		return noResults("nothing nearby")
	}
	for _, g := range results {
		if err := app.Out.Emit(Row{
			Cols:  []string{"title", "lat", "lon", "dist", "url"},
			Vals:  []string{g.Title, ftoa(g.Lat), ftoa(g.Lon), strconv.FormatFloat(g.Dist, 'f', 0, 64), g.URL},
			Value: g,
		}); err != nil {
			return err
		}
	}
	return app.Out.Flush()
}

// parseLatLon reads a coordinate from one "lat,lon" arg or two separate args.
func parseLatLon(args []string) (lat, lon float64, err error) {
	var latS, lonS string
	if len(args) == 1 {
		parts := strings.SplitN(args[0], ",", 2)
		if len(parts) != 2 {
			return 0, 0, errBadCoord
		}
		latS, lonS = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	} else {
		latS, lonS = args[0], args[1]
	}
	if lat, err = strconv.ParseFloat(latS, 64); err != nil {
		return 0, 0, errBadCoord
	}
	if lon, err = strconv.ParseFloat(lonS, 64); err != nil {
		return 0, 0, errBadCoord
	}
	return lat, lon, nil
}

func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
