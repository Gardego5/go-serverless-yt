# Go Serverless YT

Simple go lambda to retrieve and store users from a DynamoDB table.

Created by following [a youtube tutorial](https://youtu.be/zHcef4eHOc8?si=rLFL6mPqES1cxoVy)
from freeCodeCamp.org, by Akhil Sharma.

Some improvements made, mainly to reduce the number of DynamoDB requests made by
using the returned values from the `PutItem` function.

