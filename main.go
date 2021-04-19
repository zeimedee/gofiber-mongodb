package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"

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

type Employee struct {
	Id     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

const dbName = "fiber"
const MongoUrl = "mongodb://127.0.0.1:27017/" + dbName

func connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(MongoUrl))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	if err := connect(); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("go fiber")
	})

	//get all employee records
	app.Get("/employees", func(c *fiber.Ctx) error {
		query := bson.D{{}}
		cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		var employees []Employee = make([]Employee, 0)

		if err := cursor.All(c.Context(), &employees); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.JSON(employees)
	})

	// insert a new employee
	app.Post("/employees", func(c *fiber.Ctx) error {
		collection := mg.Db.Collection("employees")

		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		employee.Id = ""

		insertionResult, err := collection.InsertOne(c.Context(), employee)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
		createdRecord := collection.FindOne(c.Context(), filter)

		createdEmployee := &Employee{}
		createdRecord.Decode(createdEmployee)

		return c.Status(201).JSON(createdEmployee)
	})

	// update an employee record
	app.Put("/employees/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		employID, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return c.SendStatus(400)
		}

		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: employID}}
		update := bson.D{{Key: "$set",
			Value: bson.D{
				{Key: "name", Value: employee.Name},
				{Key: "salary", Value: employee.Salary},
				{Key: "age", Value: employee.Age},
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
		employee.Id = idParam
		return c.Status(200).JSON(employee)
	})

	//delete an employee
	app.Delete("/employees/:id", func(c *fiber.Ctx) error {
		employeeId, err := primitive.ObjectIDFromHex(
			c.Params("id"),
		)

		query := bson.D{{Key: "_id", Value: employeeId}}
		result, err := mg.Db.Collection("employees").DeleteOne(c.Context(), query)
		if err != nil {
			return c.SendStatus(500)
		}
		//if employee does not exist
		if result.DeletedCount < 1 {
			return c.SendStatus(400)
		}
		return c.SendStatus(204)
	})

	// find one employee

	app.Get("/employee/:id", func(c *fiber.Ctx) error {
		employeeId, err := primitive.ObjectIDFromHex(c.Params("id"))

		query := bson.D{{Key: "_id", Value: employeeId}}

		result := mg.Db.Collection("employees").FindOne(c.Context(), query)

		if err != nil {
			return c.Status(404).SendString("not found")
		}
		employee := &Employee{}
		result.Decode(employee)
		return c.Status(200).JSON(employee)
	})

	log.Fatal(app.Listen(":3000"))
}
