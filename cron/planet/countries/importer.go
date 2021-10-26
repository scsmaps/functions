package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	/** Load .env for local */
	godotenv.Load()

	/** Getting environment variables */
	token := os.Getenv("PLANET")
	cluster := os.Getenv("MCLUSTER")
	database := os.Getenv("MDATABASE")
	username := os.Getenv("MUSER")
	password := os.Getenv("MPASSWORD")

	/** mongoClient */
	connect := "mongodb+srv://" + username + ":" + password + "@" + cluster
	mongoClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(connect))

	if err != nil {
		log.Fatal("Connection to the database failed...")
	}

	/** restyClient */
	restyClient := resty.New()
	restyClient.SetHostURL("https://api.protectedplanet.net")
	restyClient.SetTimeout(10 * time.Second)

	/** updating countries */

	countries := struct {
		Countries []interface{} `json:"countries" bson:"countries"`
	}{}

	page := 1
	collection := mongoClient.Database(database).Collection("countries")

	for {
		log.Println("Reading page: " + strconv.Itoa(page))

		response, err := restyClient.R().
			EnableTrace().
			SetHeader("Accept", "application/json").
			SetQueryParams(map[string]string{
				"token":         token,
				"page":          strconv.Itoa(page),
				"per_page":      "50",
				"with_geometry": "true",
			}).
			Get("/v3/countries")

		if err != nil {
			log.Fatal("Error getting countries")
		}

		err = json.Unmarshal(response.Body(), &countries)
		if err != nil {
			log.Fatal("Error unmarshal json")
		}

		if len(countries.Countries) == 0 {
			break
		}

		for _, country := range countries.Countries {
			value, ok := country.(map[string]interface{})
			if !ok {
				log.Fatal("err")
			}

			opts := options.Update().SetUpsert(true)
			filter := bson.M{"planet.iso_3": value["iso_3"].(string)}

			update := bson.M{
				"$set": bson.M{
					"iso_3":   value["iso_3"].(string),
					"geojson": value["geojson"],
					"planet":  country,
				},
			}

			go collection.UpdateOne(context.TODO(), filter, update, opts)
		}

		page++
	}

	err = mongoClient.Disconnect(context.TODO())

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connection to MongoDB closed.")
}
