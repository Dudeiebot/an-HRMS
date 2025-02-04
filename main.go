package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2" // fiber take care of a lot of things for us from marshalling and unmarshalling our data
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const (
	dbName   = "fiber-hrms"
	mongoURI = "mongodb://localhost:27017/" + dbName // if i use mlab it will give me a url that i can out here as mongodb and also it depends onn the way you configure your mongodb maybe with username and password or without
)

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"` // first time using these id as a type in our struct, whhich is also binary format commonly used with MongoDB
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	) // these helps to check if your mongodb wont work
	defer cancel()

	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err
	}

	mg = MongoInstance{
		Client: client,
		Db:     db,
	}
	return nil
}

func main() {
	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	app.Get("/employee", func(c *fiber.Ctx) error {
		query := bson.D{{}}

		count, err := mg.Db.Collection("employees").CountDocuments(c.Context(), query)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		// Create a slice with the calculated capacity based on the count
		employees := make([]Employee, 0, count)

		cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		// Populate the slice with data from MongoDB
		if err := cursor.All(c.Context(), &employees); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		// Return the employees slice
		return c.JSON(employees)
	})

	app.Post("/employee", func(c *fiber.Ctx) error {
		collection := mg.Db.Collection("employees")

		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(500).SendString(err.Error())
		}
		employee.ID = ""

		insertionResult, err := collection.InsertOne(
			c.Context(),
			employee,
		) // insertone is a func that i get mongodb and it takes data and insert it into the database, just one tho
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		filter := bson.D{
			{Key: "_id", Value: insertionResult.InsertedID},
		} // just like how we created query in our app.GET route there
		createdRecord := collection.FindOne(c.Context(), filter)

		createdEmployee := &Employee{}
		createdRecord.Decode(createdEmployee)

		return c.Status(201).JSON(createdEmployee)
	})

	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		employeeID, err := primitive.ObjectIDFromHex(idParam)
		// these 2 line here is practically all the one line just as in ln 144, very easy and straightforward
		if err != nil {
			return c.SendStatus(400)
		}

		employee := new(Employee)
		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: employeeID}}
		update := bson.D{
			{
				Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}

		err = mg.Db.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(400)
			}
			return c.SendStatus(500)
		}

		employee.ID = idParam
		return c.Status(200).JSON(employee)
	})

	app.Delete("/employee/:id", func(c *fiber.Ctx) error {
		employeeID, err := primitive.ObjectIDFromHex(c.Params("id"))
		if err != nil {
			return c.Status(400).
				SendString("Invalid Employer ID")
			// check out if the id i got here is valid and the error return the necessary important shii
		}

		query := bson.D{{Key: "_id", Value: employeeID}}
		result, err := mg.Db.Collection("employees").DeleteOne(c.Context(), &query)
		if err != nil {
			return c.SendStatus(500)
		}

		if result.DeletedCount < 1 {
			return c.SendStatus(404)
		}

		return c.Status(200).JSON("record deleted")
	})

	log.Fatal(app.Listen(":3000"))
}
