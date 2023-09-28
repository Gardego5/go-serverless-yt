package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

var (
	ErrorFailedToUnmarshalRecord = "failed to unmarshal record"
	ErrorFailedToFetchRecord     = "failed to fetch record"
	ErrorInvalidUserData         = "invalid user data"
	ErrorInvalidEmail            = "invalid email"
	ErrorCouldNotMarshalItem     = "could not marshal item"
	ErrorCouldNotDeleteItem      = "could not delete item"
	ErrorCouldNotPutItem         = "could not put item"
	ErrorUserAlreadyExists       = "user already exists"
	ErrorUserDoesNotExist        = "user does not exist"
)

type User struct {
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func FetchUser(email string, tableName string, dynaClient dynamodbiface.DynamoDBAPI) (*User, error) {
	input := dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"email": {S: aws.String(email)},
		},
	}

	result, err := dynaClient.GetItem(&input)
	if err != nil {
		return nil, errors.New(ErrorFailedToFetchRecord)
	}

	item := new(User)
	err = dynamodbattribute.UnmarshalMap(result.Item, item)
	if err != nil {
		return nil, errors.New(ErrorFailedToUnmarshalRecord)
	}

	return item, nil
}

func FetchUsers(tableName string, dynaClient dynamodbiface.DynamoDBAPI) (*[]User, error) {
	input := dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}

	users := []User{}

	for {
		result, err := dynaClient.Scan(&input)

		if err != nil {
			return nil, errors.New(ErrorFailedToFetchRecord)
		}

		items := new([]User)
		err = dynamodbattribute.UnmarshalListOfMaps(result.Items, items)

		if err != nil {
			return nil, errors.New(ErrorFailedToUnmarshalRecord)
		}

		users = append(users, *items...)

		if result.LastEvaluatedKey != nil {
			input.ExclusiveStartKey = result.LastEvaluatedKey
		} else {
			break
		}
	}

	return &users, nil
}

func CreateUser(req events.APIGatewayProxyRequest, tableName string, dynaClient dynamodbiface.DynamoDBAPI) (*User, error) {
	user := User{}

	if err := json.Unmarshal([]byte(req.Body), &user); err != nil {
		return nil, errors.New(ErrorInvalidUserData)
	}

	if _, err := mail.ParseAddress(user.Email); err != nil {
		return nil, errors.New(ErrorInvalidEmail)
	}

	item, err := dynamodbattribute.MarshalMap(user)
	if err != nil {
		return nil, errors.New(ErrorCouldNotMarshalItem)
	}

	input := dynamodb.PutItemInput{
		TableName:           aws.String(tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(email)"),
	}

	if _, err := dynaClient.PutItem(&input); err != nil {
		switch err.(type) {
		case *dynamodb.ConditionalCheckFailedException:
			return nil, errors.New(ErrorUserAlreadyExists)
		default:
			fmt.Print(err)
			return nil, errors.New(ErrorCouldNotPutItem)
		}
	}

	return &user, nil
}

func UpdateUser(req events.APIGatewayProxyRequest, tableName string, dynaClient dynamodbiface.DynamoDBAPI) (*User, error) {
	user := User{}
	if err := json.Unmarshal([]byte(req.Body), &user); err != nil {
		return nil, errors.New(ErrorInvalidUserData)
	}

	item, err := dynamodbattribute.MarshalMap(user)
	if err != nil {
		return nil, errors.New(ErrorCouldNotMarshalItem)
	}

	updates := []string{}
	if user.FirstName != "" {
		updates = append(updates, updateExpressionPart("fieldName"))
	}
	if user.LastName != "" {
		updates = append(updates, updateExpressionPart("lastName"))
	}

	input := dynamodb.UpdateItemInput{
		TableName:                 aws.String(tableName),
		Key:                       item,
		UpdateExpression:          aws.String(fmt.Sprintf("SET %s", strings.Join(updates, ", "))),
		ConditionExpression:       aws.String("attribute_exists(email)"),
		ExpressionAttributeValues: item,
		ReturnValues:              aws.String("ALL_NEW"),
	}

	result, err := dynaClient.UpdateItem(&input)
	if err != nil {
		switch err.(type) {
		case *dynamodb.ConditionalCheckFailedException:
			return nil, errors.New(ErrorUserDoesNotExist)
		default:
			return nil, errors.New(ErrorCouldNotPutItem)
		}
	}

	if err := dynamodbattribute.UnmarshalMap(result.Attributes, &user); err != nil {
		return nil, errors.New(ErrorFailedToUnmarshalRecord)
	}

	return &user, nil
}

func DeleteUser(req events.APIGatewayProxyRequest, tableName string, dynaClient dynamodbiface.DynamoDBAPI) error {
	user := User{}
	if err := json.Unmarshal([]byte(req.Body), &user); err != nil {
		return errors.New(ErrorInvalidUserData)
	}

	item, err := dynamodbattribute.MarshalMap(user)
	if err != nil {
		return errors.New(ErrorCouldNotMarshalItem)
	}

	input := dynamodb.DeleteItemInput{
		TableName:           aws.String(tableName),
		Key:                 item,
		ConditionExpression: aws.String("attribute_exists(email)"),
	}

	if _, err := dynaClient.DeleteItem(&input); err != nil {
		switch err.(type) {
		case *dynamodb.ConditionalCheckFailedException:
			return errors.New(ErrorUserDoesNotExist)
		default:
			return errors.New(ErrorCouldNotDeleteItem)
		}
	}

	return nil
}

func updateExpressionPart(fieldName string) string {
	return fmt.Sprintf("%s = :%s", fieldName, fieldName)
}
