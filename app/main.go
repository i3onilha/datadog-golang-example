package main

import (
	"context"
	"log"
	"os"
	"time"

	gintrace "github.com/DataDog/dd-trace-go/contrib/gin-gonic/gin/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// User represents a user document in MongoDB
type User struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name      string             `json:"name" bson:"name"`
	Email     string             `json:"email" bson:"email"`
	Age       int                `json:"age" bson:"age"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Age   int    `json:"age" binding:"required,min=1,max=150"`
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email" binding:"omitempty,email"`
	Age   int    `json:"age" binding:"omitempty,min=1,max=150"`
}

var (
	client     *mongo.Client
	collection *mongo.Collection
)

func initDB() {
	// Get MongoDB connection string from environment or use default
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		// Default connection string for docker-compose setup
		mongoUser := os.Getenv("MONGO_USER")
		mongoPass := os.Getenv("MONGO_PASSWORD")
		if mongoUser == "" {
			mongoUser = "root"
		}
		if mongoPass == "" {
			mongoPass = "password"
		}
		mongoHost := os.Getenv("MONGO_HOST")
		if mongoHost == "" {
			mongoHost = "mongodb"
		}
		mongoURI = "mongodb://" + mongoUser + ":" + mongoPass + "@" + mongoHost + ":27017/?authSource=admin"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Ping the database
	if err = client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	log.Println("Connected to MongoDB successfully")

	// Get collection
	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "go_api_demo"
	}
	collection = client.Database(dbName).Collection("users")
}

func main() {
	// Start Datadog tracer
	tracer.Start(
		tracer.WithService("go-api-demo"),
		tracer.WithEnv("dev"),
		tracer.WithServiceVersion("1.0.0"),
	)
	defer tracer.Stop()

	// Initialize MongoDB connection
	initDB()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}()

	// Create a Gin router
	r := gin.Default()

	// Add DataDog tracing middleware
	r.Use(gintrace.Middleware("go-api-demo"))

	// Health check endpoint
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// CRUD endpoints
	api := r.Group("/api/v1")
	{
		// Create a new user
		api.POST("/users", createUser)

		// Get all users
		api.GET("/users", getUsers)

		// Get a user by ID
		api.GET("/users/:id", getUserByID)

		// Update a user by ID
		api.PUT("/users/:id", updateUser)

		// Delete a user by ID
		api.DELETE("/users/:id", deleteUser)
	}

	log.Println("Server running on :8080")
	r.Run(":8080")
}

// createUser creates a new user in MongoDB
func createUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	user := User{
		ID:        primitive.NewObjectID(),
		Name:      req.Name,
		Email:     req.Email,
		Age:       req.Age,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, user)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create user: " + err.Error()})
		return
	}

	user.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(201, user)
}

// getUsers retrieves all users from MongoDB
func getUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch users: " + err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var users []User
	if err = cursor.All(ctx, &users); err != nil {
		c.JSON(500, gin.H{"error": "Failed to decode users: " + err.Error()})
		return
	}

	if users == nil {
		users = []User{}
	}

	c.JSON(200, gin.H{"users": users, "count": len(users)})
}

// getUserByID retrieves a user by ID from MongoDB
func getUserByID(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(404, gin.H{"error": "User not found"})
			return
		}
		c.JSON(500, gin.H{"error": "Failed to fetch user: " + err.Error()})
		return
	}

	c.JSON(200, user)
}

// updateUser updates a user by ID in MongoDB
func updateUser(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid user ID"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Build update document
	update := bson.M{
		"updated_at": time.Now(),
	}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if req.Email != "" {
		update["email"] = req.Email
	}
	if req.Age > 0 {
		update["age"] = req.Age
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to update user: " + err.Error()})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}

	// Fetch and return updated user
	var user User
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch updated user: " + err.Error()})
		return
	}

	c.JSON(200, user)
}

// deleteUser deletes a user by ID from MongoDB
func deleteUser(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to delete user: " + err.Error()})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}

	c.JSON(200, gin.H{"message": "User deleted successfully"})
}
