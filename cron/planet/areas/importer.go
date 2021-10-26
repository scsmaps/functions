package main

import (
	"context"
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

type Response struct {
	Areas []interface{} `json:"protected_areas" bson:"protected_areas"`
}

func main() {
	/** Load .env for local */
	godotenv.Load()

	/** Getting environment variables */
	token := os.Getenv("PLANET")
	cluster := os.Getenv("MCLUSTER")
	database := os.Getenv("MDATABASE")
	username := os.Getenv("MUSER")
	password := os.Getenv("MPASSWORD")

	/** restyClient */
	restyClient := resty.New()
	restyClient.SetTimeout(10 * time.Second)

	/** mongoClient */
	connect := "mongodb+srv://" + username + ":" + password + "@" + cluster
	clientOptions := options.Client().ApplyURI(connect)
	mongoClient, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal("Connection to the database failed...")
	}

	countries := mongoClient.Database(database).Collection("countries")
	cursor, err := countries.Find(context.TODO(), bson.M{})
	if err != nil {
		panic(err)
	}

	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	collection := mongoClient.Database(database).Collection("areas")

	for _, result := range results {
		log.Println(result["iso_3"])
		count := 1
		for {
			log.Println("Reading page: " + strconv.Itoa(count))

			response := Response{}
			_, err := restyClient.R().
				EnableTrace().
				SetHeader("Accept", "application/json").
				SetResult(&response).
				SetQueryParams(map[string]string{
					"token":         token,
					"with_geometry": "true",
					"marine":        "true",
					"country":       result["iso_3"].(string),
					"page":          strconv.Itoa(count),
					"per_page":      "50",
				}).
				Get("https://api.protectedplanet.net/v3/protected_areas/search")

			/** Some pages will time out, just skip them */
			if err != nil {
				log.Println("Skipping")
				count++
				continue
			}

			if len(response.Areas) == 0 {
				log.Println("All done")
				break
			}

			for _, element := range response.Areas {
				v, ok := element.(map[string]interface{})
				if !ok {
					log.Fatal("err")
				}

				if bool(v["marine"].(bool)) == true {
					opts := options.Update().SetUpsert(true)
					filter := bson.M{"planet.id": int(v["id"].(float64))}

					update := bson.M{
						"$set": bson.M{
							"geojson": v["geojson"],
							"planet":  element,
						},
					}

					go collection.UpdateOne(context.TODO(), filter, update, opts)
				}
			}

			log.Println("Done")
			count++
		}
	}

	err = mongoClient.Disconnect(context.TODO())

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connection to MongoDB closed.")
}
