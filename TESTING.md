# MongoDB Collection Acceptance Tests

This directory contains acceptance tests for the `mongodb_db_collection` resource.

## Running the Tests

### Prerequisites

1. **MongoDB Instance**: You need a running MongoDB instance. You can use the provided Docker Compose setup:
   ```bash
   cd docker
   docker-compose up -d
   ```

2. **Environment Variables**: Set the following environment variables for the test connection:
   ```bash
   export MONGO_HOST=127.0.0.1
   export MONGO_PORT=27017
   export MONGO_USR=root
   export MONGO_PWD=root
   export MONGO_AUTH_DB=admin
   export TF_ACC=1
   ```

### Run Collection Tests

```bash
# Run all collection tests
go test ./mongodb -v -run TestAccMongoDBCollection

# Run a specific test
go test ./mongodb -v -run TestAccMongoDBCollection_Basic
```

## Test Cases

The test suite includes the following test cases:

1. **TestAccMongoDBCollection_Basic**: Tests basic collection creation and destruction
2. **TestAccMongoDBCollection_WithChangeStreamImages**: Tests collection with change stream pre/post images enabled
3. **TestAccMongoDBCollection_Update**: Tests updating collection configuration (changing change stream settings)

## Test Configuration

Each test:
- Creates a collection with a random name and database
- Verifies the collection exists in MongoDB
- Checks that all attributes are set correctly
- Ensures the collection is properly destroyed after the test

The tests use the Terraform Plugin SDK v2 testing framework and follow standard Terraform provider testing patterns.