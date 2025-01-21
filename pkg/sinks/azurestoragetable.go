package sinks

import (
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/resmoio/kubernetes-event-exporter/pkg/kube"
)

type AzureStorageTableConfig struct {
	// Connection specific
	ConnectionString string                 `yaml:"connectionString"`
	TableName        string                 `yaml:"tableName"`
	TableNameFormat  string                 `yaml:"tableNameFormat"`
	Layout           map[string]interface{} `yaml:"layout"`
}

type AzureStorageTable struct {
	serviceClient *aztables.ServiceClient
	cfg           *AzureStorageTableConfig

	lastTableName string
}

func NewAzureStorageTable(cfg *AzureStorageTableConfig) (*AzureStorageTable, error) {
	sertviceClient, err := aztables.NewServiceClientFromConnectionString(cfg.ConnectionString, nil)
	if err != nil {
		return nil, err
	}

	return &AzureStorageTable{
		serviceClient: sertviceClient,
		cfg:           cfg,
	}, nil
}

func (a *AzureStorageTable) formatTableName() string {
	if a.cfg.TableNameFormat != "" {
		now := time.Now().UTC()
		return now.Format(a.cfg.TableNameFormat)
	}
	return a.cfg.TableName
}

func (a *AzureStorageTable) Send(ctx context.Context, ev *kube.EnhancedEvent) error {
	tableName := a.formatTableName()
	if tableName != a.lastTableName {
		// Do not send create request each time,
		// only when table name changed
		_, err := a.serviceClient.CreateTable(ctx, tableName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) {
				if respErr.StatusCode == 409 {
					// Table already exists
					err = nil
				}
			} else {
				return err
			}
		}
		a.lastTableName = tableName
	}

	// Storage table doesn't allow dots in column names
	de := ev.DeDot()
	ev = &de

	jsonEv := ev.ToJSON()
	var dictEv map[string]interface{}
	json.Unmarshal(jsonEv, &dictEv)

	// We need to flatten data since azure storage table doesn't support nested data
	flattened, err := a.flatten(dictEv)
	if err != nil {
		return err
	}

	entity := aztables.EDMEntity{
		Entity: aztables.Entity{
			PartitionKey: ev.Namespace,
			RowKey:       string(ev.UID),
		},
		Properties: flattened,
	}

	marshalled, err := json.Marshal(entity)
	if err != nil {
		return err
	}

	client := a.serviceClient.NewClient(tableName)
	_, err = client.AddEntity(context.TODO(), marshalled, nil)
	// if err != nil {
	// 	var respErr *azcore.ResponseError
	// 	if errors.As(err, &respErr) {
	// 		bytes, err := io.ReadAll(respErr.RawResponse.Body)
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}

	// 		log.Println(string(bytes))
	// 	}
	// }
	return err
}

type dictPair struct {
	key string
	val interface{}
}

func (a *AzureStorageTable) flatten(input map[string]interface{}) (map[string]interface{}, error) {
	output := make(map[string]interface{})

	var stack = list.New()
	stack.PushBack(dictPair{
		key: "",
		val: input,
	})

	for {
		element := stack.Back()
		if element == nil {
			break
		}
		stack.Remove(element)

		pair, ok := element.Value.(dictPair)
		if !ok {
			return output, errors.New("invalid stack element")
		}

		switch v := pair.val.(type) {
		case map[string]interface{}:
			for key, val := range v {
				stack.PushBack(dictPair{
					key: strings.TrimLeft(pair.key+"_"+key, "_"),
					val: val,
				})
			}
		case []interface{}:
			for i, val := range v {
				stack.PushBack(dictPair{
					key: strings.TrimLeft(pair.key+"_"+strconv.Itoa(i), "_"),
					val: val,
				})
			}
		default:
			r := strings.NewReplacer(
				"/", "_",
				"\\", "_",
				"#", "_",
				"?", "_",
				"\t", "_",
				"\n", "_",
				"\r", "_",
				"-", "_",
			)
			key := r.Replace(pair.key)
			firstChar := key[0]
			if firstChar >= '0' && firstChar <= '9' {
				key = "_" + key
			}
			output[key] = v
		}

	}
	return output, nil
}

func (a *AzureStorageTable) Close() {
	// No-op
}
