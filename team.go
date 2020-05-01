package main

import (
	"fmt"
	"time"

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

type teamData struct {
	date       time.Time
	cycleTime  float64
	throughput float64
	comment    string
}
type team struct {
	name string
	data []teamData
}

func (t *team) renderer() chart.Chart {

	xval := []float64{}
	cycleYval := []float64{}
	throughputYval := []float64{}
	comments := []chart.Value2{}

	for i := range t.data {
		date := float64(t.data[i].date.Unix())
		xval = append(xval, date)
		cycleYval = append(cycleYval, t.data[i].cycleTime)
		throughputYval = append(throughputYval, t.data[i].throughput)
		if t.data[i].comment != "" {
			comments = append(comments, chart.Value2{
				XValue: date,
				YValue: t.data[i].cycleTime,
				Label:  t.data[i].comment,
			})
			comments = append(comments, chart.Value2{
				XValue: date,
				YValue: t.data[i].throughput,
				Label:  t.data[i].comment,
			})
		}
	}

	graph := chart.Chart{
		Background: chart.Style{ClassName: "background"},
		Canvas: chart.Style{
			ClassName: "canvas",
		},
		XAxis: chart.XAxis{
			Name:         "Date",
			TickPosition: chart.TickPositionBetweenTicks,
			ValueFormatter: func(v interface{}) string {
				typed := v.(float64)
				typedDate := time.Unix(int64(typed), 0)
				return fmt.Sprintf("%d-%d-%d", typedDate.Year(), typedDate.Month(), typedDate.Day())
			},
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				Name: "Cycle time (days)    ",
				Style: chart.Style{
					StrokeColor: drawing.ColorFromHex("f27713"),
					StrokeWidth: 2,
				},
				XValues: xval,
				YValues: cycleYval,
			},
			chart.ContinuousSeries{
				Name: " Throughput (# of tasks)    ",
				Style: chart.Style{
					StrokeColor: drawing.ColorFromHex("2a317c"),
					StrokeWidth: 2,
				},
				XValues: xval,
				YValues: throughputYval,
			},
			chart.AnnotationSeries{
				Annotations: comments,
			},
		},
	}

	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}

	return graph
}

func newTeam(name string) *team {
	return &team{
		name: name,
	}
}
