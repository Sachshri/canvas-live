package config

type Config struct {
}

type MongoConfigStruct struct {
	MongoUri                      string
	DatabaseName                  string
	UserCollectionName            string
	DocumentCollectionName        string
	SharedDocRecordCollectionName string
}

var MongoConfig = MongoConfigStruct{
	MongoUri:                      "mongodb://canvas-live-mongodb:27017",
	DatabaseName:                  "default",
	UserCollectionName:            "user",
	DocumentCollectionName:        "document",
	SharedDocRecordCollectionName: "shared",
}
