// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package metricset

import (
	"errors"
	"fmt"
	"time"

	"github.com/santhosh-tekuri/jsonschema"

	"github.com/elastic/apm-server/model/metricset/generated/schema"
	"github.com/elastic/apm-server/transform"
	"github.com/elastic/apm-server/utility"
	"github.com/elastic/apm-server/validation"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

const (
	processorName = "metric"
	docType       = "metric"
)

var (
	Metrics         = monitoring.Default.NewRegistry("apm-server.processor.metric", monitoring.PublishExpvar)
	transformations = monitoring.NewInt(Metrics, "transformations")
	processorEntry  = common.MapStr{"name": processorName, "event": docType}
)

var cachedModelSchema = validation.CreateSchema(schema.ModelSchema, "metricset")

func ModelSchema() *jsonschema.Schema {
	return cachedModelSchema
}

type Sample struct {
	Name  string
	Value float64
}

type Metricset struct {
	Samples   []*Sample
	Labels    common.MapStr
	Timestamp time.Time
}

type metricsetDecoder struct {
	*utility.ManualDecoder
}

func DecodeEvent(input interface{}, err error) (transform.Transformable, error) {
	if err != nil {
		return nil, err
	}
	if input == nil {
		return nil, errors.New("no data for metric event")
	}
	raw, ok := input.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid type for metric event")
	}

	md := metricsetDecoder{&utility.ManualDecoder{}}
	e := Metricset{
		Samples:   md.decodeSamples(raw["samples"]),
		Timestamp: md.TimeEpochMicro(raw, "timestamp"),
	}

	if md.Err != nil {
		return nil, md.Err
	}

	if tags := utility.Prune(md.MapStr(raw, "tags")); len(tags) > 0 {
		e.Labels = tags
	}

	return &e, nil
}

func (md *metricsetDecoder) decodeSamples(input interface{}) []*Sample {
	if input == nil {
		md.Err = errors.New("no samples for metric event")
		return nil
	}
	raw, ok := input.(map[string]interface{})
	if !ok {
		md.Err = errors.New("invalid type for samples in metric event")
		return nil
	}

	samples := make([]*Sample, len(raw))
	i := 0
	for name, s := range raw {
		if s == nil {
			continue
		}
		sampleMap, ok := s.(map[string]interface{})
		if !ok {
			md.Err = fmt.Errorf("invalid sample: %s: %s", name, s)
			return nil
		}

		samples[i] = &Sample{
			Name:  name,
			Value: md.Float64(sampleMap, "value"),
		}
		if md.Err != nil {
			return nil
		}
		i++
	}
	return samples
}

func (me *Metricset) Transform(tctx *transform.Context) []beat.Event {
	transformations.Inc()
	if me == nil {
		return nil
	}

	fields := common.MapStr{}
	for _, sample := range me.Samples {
		if _, err := fields.Put(sample.Name, sample.Value); err != nil {
			logp.NewLogger("transform").Warnf("failed to transform sample %#v", sample)
			continue
		}
	}

	fields["processor"] = processorEntry
	if me.Labels != nil {
		// normalize map entries by calling utility.Add
		utility.Add(fields, "labels", me.Labels)
	}
	tctx.Metadata.Merge(fields)

	if me.Timestamp.IsZero() {
		me.Timestamp = tctx.RequestTime
	}

	return []beat.Event{
		{
			Fields:    fields,
			Timestamp: me.Timestamp,
		},
	}
}
