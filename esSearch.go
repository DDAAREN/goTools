package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"

	"gopkg.in/olivere/elastic.v5"
)

type LxLog struct {
	Cid       int     `json:"cid"`
	Rid       int     `json:"rid"`
	Prpc      string  `json:"prpc"`
	LogFile   string  `json:"log_file"`
	Rpc       string  `json:"rpc"`
	File      string  `json:"file"`
	Pod       string  `json:"pod"`
	Line      int     `json:"line"`
	LogLevel  string  `json:"log_level"`
	Category  string  `json:"cat"`
	ReqId     string  `json:"reqid"`
	Tc        float64 `json:"tc"`
	LogTime   string  `json:"log_time"`
	LogHost   string  `json:"log_host"`
	Msg       string  `json:"msg"`
	LogServer string  `json:"log_server"`
	Platform  string
}

func SearchES() []LxLog {
	//todayStr := time.Now().Format("2006.01.02")
	yesterdayStr := time.Now().AddDate(0, 0, -1).Format("2006.01.02")
	//ctx := context.Background()
	ctx := context.TODO()
	client, err := elastic.NewClient(
	//elastic.SetURL(
	//	"127.0.0.1:9200"),
	)
	if err != nil {
		panic(err)
	}
	defer client.Stop()

	tmpQuery := elastic.NewBoolQuery()
	tmpQuery = tmpQuery.Must(
		elastic.NewQueryStringQuery("ns: nsName AND (log_level: ERROR OR log_level: FATAL)"),
	)
	//searchResult, err := client.Search().
	//	Index("indexName-"+todayStr).
	//	Query(tmpQuery).
	//	Sort("log_time", false).
	//	From(0).Size(10000).
	//	Do(ctx)
	//search := client.Scroll("indexName-"+todayStr).
	search := client.Scroll("indexName-" + yesterdayStr).
		Query(tmpQuery)
		//Sort("log_time", false)
		//Size(10000)

	var result []LxLog
	var searchResult *elastic.SearchResult
	totalHits := int64(0)

	for {
		searchResult, err = search.Do(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		totalHits = searchResult.TotalHits()
		for _, hit := range searchResult.Hits.Hits {
			var t LxLog
			err := json.Unmarshal(*hit.Source, &t)
			if err != nil {
				fmt.Printf("%s || %+v\n", err.Error(), string(*hit.Source))
				continue
			}
			t.Platform = "platform"
			result = append(result, t)
		}
	}
	fmt.Printf("Total Find %d Error logs\n", totalHits)
	return result
}

type logExample struct {
	Example LxLog
	Count   int
}

type Entity struct {
	Key     string
	Count   int
	Example LxLog
}

type Entities []Entity

func (e Entities) Len() int { return len(e) }
func (e Entities) Less(i, j int) bool {
	return e[i].Count > e[j].Count
}
func (e Entities) Swap(i, j int) { e[i], e[j] = e[j], e[i] }

func main() {
	allErrorLogs := SearchES()
	tmpSet := make(map[string]*logExample)
	for _, l := range allErrorLogs {
		key := l.File + "-" + strconv.Itoa(l.Line)
		if _, in := tmpSet[key]; in {
			tmpSet[key].Count++
			continue
		}
		tmpSet[key] = &logExample{
			Example: l,
			Count:   1,
		}
	}

	entities := Entities{}
	for k, v := range tmpSet {
		entities = append(entities, Entity{
			Key:     k,
			Example: v.Example,
			Count:   v.Count,
		})
	}
	sort.Sort(entities)

	js := json.NewEncoder(os.Stdout)
	//js.SetIndent("  ", "  ")
	for _, e := range entities {
		js.Encode(e)
	}
}
