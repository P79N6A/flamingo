package goredis

import (
	"errors"
	"time"

	"code.byted.org/kv/redis-v6"
)

type Pipeline struct {
	redis.Pipeliner
	cluster            string
	psm                string
	metricsServiceName string
	name               string
}

// func (p *Pipeline) SetPSM

// Exec executes all previously queued commands using one client-server roundtrip.
//
// Exec returns list of commands and error.
// miss is not a error in pipeline.
// you should use Cmder.Err() == redis.Nil to find whether miss occur or not
//
// After Commit, you should use Close() to close the pipeline releasing open resources.
func (p *Pipeline) Exec() ([]redis.Cmder, error) {
	start := time.Now().UnixNano()

	cmder, _ := p.Pipeliner.Exec()
	var status string = CALLSTATUS_SUCCESS
	var resErr error
	var isEmpty = false
	latency := (time.Now().UnixNano() - start) / 1000
	pipelineCmdNum := len(cmder)

	if pipelineCmdNum == 0 {
		resErr = errors.New("pipeline cmd num is 0")
		isEmpty = true
	}

	cmdThroughputCounter := make(map[string]int)
	cmdErrorCounter := make(map[string]int)
	cmdSuccessCounter := make(map[string]int)
	for _, res := range cmder {
		cmdStr := res.Name()
		counter, ok := cmdThroughputCounter[cmdStr]
		if ok {
			cmdThroughputCounter[cmdStr] = counter + 1
		} else {
			cmdThroughputCounter[cmdStr] = 1
		}
		// one or more miss occur in pipeline, and we think miss is not a error in pipeline
		if res.Err() != nil && res.Err() != redis.Nil {
			counter, ok := cmdErrorCounter[cmdStr]
			if ok {
				cmdErrorCounter[cmdStr] = counter + 1
			} else {
				cmdErrorCounter[cmdStr] = 1
			}
			if resErr == nil {
				resErr = res.Err()
			}
		} else {
			counter, ok := cmdSuccessCounter[cmdStr]
			if ok {
				cmdSuccessCounter[cmdStr] = counter + 1
			} else {
				cmdSuccessCounter[cmdStr] = 1
			}

		}
	}
	// Aggregate pipeline cmd metrics by cmdStr
	// separate cmd old
	for cmdStr, counter := range cmdThroughputCounter {
		cmdTags := map[string]string{
			"cluster":       p.cluster,
			"caller":        p.psm,
			"cmd":           cmdStr,
			"lang":          "go",
			"pipeline_name": p.name,
		}
		metricsClient.EmitCounter("throughput", counter, "", cmdTags)
	}
	for cmdStr, counter := range cmdErrorCounter {
		cmdTags := map[string]string{
			"cluster":       p.cluster,
			"caller":        p.psm,
			"cmd":           cmdStr,
			"lang":          "go",
			"pipeline_name": p.name,
		}
		metricsClient.EmitCounter("error", counter, "", cmdTags)
	}
	// separate cmd new
	for cmdStr, counter := range cmdSuccessCounter {
		cmdTags := map[string]string{
			"cluster":       p.cluster,
			"method":        cmdStr,
			"to":            p.metricsServiceName,
			"pipeline_name": p.name,
			"from_cluster":  "default",
			"to_cluster":    "default",
		}
		metricsClientWithPsm.EmitCounter("call."+status+".throughput", counter, "", cmdTags)
	}

	// pipeline cmd old
	pipelineTags := map[string]string{
		"cluster":       p.cluster,
		"caller":        p.psm,
		"cmd":           "pipeline",
		"lang":          "go",
		"pipeline_name": p.name,
	}
	metricsClient.EmitTimer("latency", latency, "", pipelineTags)
	metricsClient.EmitCounter("throughput", 1, "", pipelineTags)
	if resErr != nil && !isEmpty {
		metricsClient.EmitCounter("error", 1, "", pipelineTags)
	}
	if pipelineCmdNum > 500 {
		metricsClient.EmitCounter("big_pipeline", 1, "", pipelineTags)
	}

	// pipeline cmd new
	pipelineTagsForPsmClient := map[string]string{
		"cluster":       p.cluster,
		"method":        "pipeline",
		"to":            p.metricsServiceName,
		"pipeline_name": p.name,
		"from_cluster":  "default",
		"to_cluster":    "default",
	}
	metricsClientWithPsm.EmitTimer("call."+status+".latency.us", latency, "", pipelineTagsForPsmClient)

	if resErr != nil && !isEmpty {
		status = CALLSTATUS_ERROR
	}
	metricsClientWithPsm.EmitCounter("call."+status+".throughput", 1, "", pipelineTagsForPsmClient)

	return cmder, resErr
}
