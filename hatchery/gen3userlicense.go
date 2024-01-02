package hatchery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/google/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ErrNoLicenseIds = errors.New("no license ids available")

var initializeDbConfig = func() *DbConfig {
	// Create a new dynamoDB client
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String("us-east-1"),
			// Use this endpoint for running locally
			// Endpoint: aws.String("http://localhost:8000"),
		},
	}))
	return &DbConfig{
		DynamoDb: dynamodb.New(sess),
	}
}

var getActiveGen3UserLicenses = func(dbconfig *DbConfig) (gen3UserLicenses *[]Gen3UserLicense, err error) {
	// Query the table to get all active user license items

	targetEnvironment := os.Getenv("GEN3_ENDPOINT")
	// Maybe also put the global secondary index name in config
	Config.Logger.Printf("Ready to query table for active users: %s", Config.Config.Gen3UserLicenseTable)
	Config.Logger.Printf("Environment = %s", targetEnvironment)

	// TODO: filter by license-type
	keyEx1 := expression.Key("environment").Equal(expression.Value(aws.String(targetEnvironment)))
	keyEx2 := expression.Key("isActive").Equal(expression.Value("True"))
	expr, err := expression.NewBuilder().WithKeyCondition(expression.KeyAnd(keyEx1, keyEx2)).Build()
	if err != nil {
		Config.Logger.Printf("Error in building expression for query: %s", err)
		return nil, err
	}
	res, err := dbconfig.DynamoDb.Query(&dynamodb.QueryInput{
		TableName:                 aws.String(Config.Config.Gen3UserLicenseTable),
		IndexName:                 aws.String("activeUsersIndex"),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
	})
	if err != nil {
		Config.Logger.Printf("Error in active user query: %s", err)
		return nil, err
	}
	fmt.Println(res.Items)

	var userLicenses []Gen3UserLicense
	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &userLicenses)
	if err != nil {
		Config.Logger.Printf("Error in unmarshalling active users: %s", err)
		return nil, err
	}

	fmt.Println(userLicenses)
	return &userLicenses, nil
}

func getNextLicenseId(activeGen3UserLicenses *[]Gen3UserLicense, maxLicenseIds int) int {
	// Determine the next available licenseId [1..6], return 0 if no ids
	if len(*activeGen3UserLicenses) == 0 {
		return 1
	}
	var idInUsedIds bool
	for i := 1; i <= maxLicenseIds; i++ {
		idInUsedIds = false
		for _, v := range *activeGen3UserLicenses {
			if i == v.LicenseId {
				idInUsedIds = true
				break
			}
		}
		if !idInUsedIds {
			Config.Logger.Printf("Next available license id: %d", i)
			return i
		}
	}
	// No ids available
	return 0
}

var createGen3UserLicense = func(dbconfig *DbConfig, userId string, licenseId int) (gen3UserLicense Gen3UserLicense, err error) {
	// Create a new user-license object and put in table

	targetEnvironment := os.Getenv("GEN3_ENDPOINT")
	// Maybe also put the global secondary index name in config
	Config.Logger.Printf("Ready to put item for new user license in table: %s", Config.Config.Gen3UserLicenseTable)
	Config.Logger.Printf("Environment = %s", targetEnvironment)

	itemId := uuid.New().String()
	currentUnixTime := int(time.Now().Unix())

	// create new Gen3UserLicense
	newItem := Gen3UserLicense{}
	newItem.LicenseType = Config.Config.Gen3LicenseType
	newItem.ItemId = itemId
	newItem.Environment = targetEnvironment
	newItem.UserId = userId
	newItem.LicenseId = licenseId
	newItem.IsActive = "True"
	newItem.FirstUsedTimestamp = currentUnixTime
	newItem.LastUsedTimestamp = currentUnixTime

	// marshall Gen3UserLicense into dynamodb item
	item, err := dynamodbattribute.MarshalMap(newItem)
	if err != nil {
		Config.Logger.Printf("Error: could not marshal new item: %s", err)
		return newItem, err
	}
	// put item
	_, err = dbconfig.DynamoDb.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(Config.Config.Gen3UserLicenseTable),
		Item:      item,
	})
	if err != nil {
		Config.Logger.Printf("Error: could not add item to table: %s", err)
		return newItem, err
	}
	Config.Logger.Printf("Debug: newItem submitted to table: %v", newItem)
	// Return the new gen3-user-license item that we created; putItem does not return new items.
	return newItem, nil
}

var setGen3UserLicenseInactive = func(dbconfig *DbConfig, itemId string) (Gen3UserLicense, error) {
	// Update an item to mark as inactive

	targetEnvironment := os.Getenv("GEN3_ENDPOINT")
	// Maybe also put the global secondary index name in config
	Config.Logger.Printf("Ready to update existing user license in table: %s", Config.Config.Gen3UserLicenseTable)
	Config.Logger.Printf("Environment = %s", targetEnvironment)

	isActive := "False"
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":active": {
				S: aws.String(isActive),
			},
		},
		TableName: aws.String(Config.Config.Gen3UserLicenseTable),
		// Use the composite primary key: itemId, environment
		Key: map[string]*dynamodb.AttributeValue{
			"itemId": {
				S: aws.String(itemId),
			},
			"environment": {
				S: aws.String(targetEnvironment),
			},
		},
		// AWS docs are bad: 'UPDATED_NEW' is not an accepted value.
		// Allowable values are 'NONE' or 'ALL_OLD' and no new values are returned.
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set isActive = :active"),
	}

	res, err := dbconfig.DynamoDb.UpdateItem(input)
	if err != nil {
		Config.Logger.Printf("Error: could not update item in table: %s", err)
		return Gen3UserLicense{}, err
	}

	var updatedItem Gen3UserLicense
	err = dynamodbattribute.UnmarshalMap(res.Attributes, &updatedItem)
	if err != nil {
		Config.Logger.Printf("Error: could not unmarshal updated item: %s", err)
		return Gen3UserLicense{}, err
	}

	Config.Logger.Printf("Debug: updatedItem submitted to table: %v", updatedItem)
	return updatedItem, nil

}

var getKubeClientSet = func() (clientset kubernetes.Interface, err error) {
	// Get the kubernetes client set
	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"
	if _, err := os.Stat(kubeConfigPath); err == nil {
		// out of cluster, eg local
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			Config.Logger.Printf("Error: Could not build config for out of cluster client, %s", err)
			return nil, err
		}
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			Config.Logger.Printf("Error: Could not create clientset for out of cluster client, %s", err)
			return nil, err
		}
	} else {
		// in cluster
		config, err := rest.InClusterConfig()
		if err != nil {
			Config.Logger.Printf("Error: Could not build config for in cluster client, %s", err)
			return nil, err
		}
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			Config.Logger.Printf("Error: Could not create clientset for in cluster client, %s", err)
			return nil, err
		}
	}

	return clientset, nil

}

var getLicenseFromKubernetes = func(clientset kubernetes.Interface) (licenseString string, err error) {
	// Read the gen3-license string from the g3auto kubernetes secret
	g3autoName := Config.Config.Gen3G3autoName
	g3autoKey := Config.Config.Gen3G3autoKey

	var namespace string
	var ok bool
	if namespace, ok = Config.Config.Sidecar.Env["NAMESPACE"]; ok {
		Config.Logger.Printf("Searching configured namespace for g3auto secret: %s", namespace)
	} else {
		Config.Logger.Printf("Error: namespace is not configured. Will try 'default'")
		namespace = "default"
	}

	//var secretsClient coreV1Types.SecretInterface
	secretsClient := clientset.CoreV1().Secrets(namespace)
	secret, err := secretsClient.Get(context.TODO(), g3autoName, metaV1.GetOptions{})
	if err != nil {
		Config.Logger.Printf("Error: could not get secret from kubernetes: %s", err)
		return "", err
	}
	licenseString = string(secret.Data[g3autoKey])
	// some g3auto secrets may have multiple strings separated by newlines
	licenseString = strings.Split(licenseString, "\n")[0]

	return licenseString, nil

}
