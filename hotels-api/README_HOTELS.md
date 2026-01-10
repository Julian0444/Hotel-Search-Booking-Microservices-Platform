# Hotels API - Documentation

## 📋 Overview

The Hotels API is a microservice designed to manage hotel information and reservations in a distributed system. It implements a layered architecture with caching, database persistence, and event-driven communication.

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     HTTP Handlers                           │
│             (Gin Controllers + JWT middlewares)             │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                    Service Layer                            │
│              (Business Logic & Orchestration)               │
│  - Cache-aside pattern implementation                       │
│  - DAO ↔ Domain conversions                                 │
│  - Event publishing (RabbitMQ) for hotel CRUD               │
└─────────────┬───────────────────────┬───────────────────────┘
              │                       │
    ┌─────────▼────────┐    ┌────────▼─────────┐
    │  Cache Repository│    │  Main Repository │
    │ (ccache in-memory)    │    (MongoDB)     │
    └──────────────────┘    └──────────────────┘
```

## 🎯 Core Features

### Hotel Management
- ✅ Create, Read, Update, Delete (CRUD) operations
- ✅ In-memory caching (ccache) with TTL (cache-aside)
- ✅ Availability checking across multiple hotels
- ✅ Concurrent operations using goroutines

### Reservation System
- ✅ Reservation lifecycle management
- ✅ User-specific reservation queries
- ✅ Hotel availability calculation based on active reservations
- ✅ Batch operations for hotel cleanup

### Performance Optimization
- ✅ Cache-aside pattern for read operations
- ✅ Concurrent availability checking
- ✅ Recommended MongoDB indexes (manual)

---

## 🌐 HTTP API Endpoints

All routes are implemented in `cmd/main.go` using Gin.

### Public
- `GET /health`
- `GET /hotels/:hotel_id`
- `GET /hotels/:hotel_id/reservations`
- `POST /hotels/availability`

### Authenticated user (JWT required)
Requires `Authorization: Bearer <token>` with claims:
- `user_id`: user identifier (number or string)
- `tipo`: user role (e.g. `cliente`, `administrador`)

Routes:
- `POST /reservations`
- `DELETE /reservations/:id`
- `GET /users/:user_id/reservations`
- `GET /users/:user_id/hotels/:hotel_id/reservations`

### Admin only (JWT required)
Requires `tipo=administrador`.

Routes:
- `POST /admin/hotels`
- `PUT /admin/hotels/:hotel_id`
- `DELETE /admin/hotels/:hotel_id`
- `GET /admin/microservices`
- `POST /admin/microservices/scale`
- `GET /admin/microservices/:service_name/logs`
- `POST /admin/microservices/:service_name/restart`

---

## 📦 Data Models

### Hotel (DAO Layer)
```go
type Hotel struct {
    ID            string    // Unique identifier
    Name          string    // Hotel name
    Description   string    // Detailed description
    Address       string    // Street address
    City          string    // City location
    Country       string    // Country
    Rating        float64   // Average rating (0-5)
    PricePerNight float64   // Price per night in USD
    AvaiableRooms int       // Total available rooms
    Amenities     []string  // List of amenities (WiFi, Pool, etc.)
}
```

### Reservation (DAO Layer)
```go
type Reservation struct {
    ID        string    // Unique identifier
    HotelID   string    // Reference to hotel
    HotelName string    // Denormalized hotel name
    UserID    string    // User who made the reservation
    CheckIn   time.Time // Check-in date
    CheckOut  time.Time // Check-out date
}
```

### Domain Models
Domain models mirror DAO models but may include additional business logic fields and validation rules.

---

## 🔧 API Functions

### Service Layer (`internal/services/hotels_service.go`)

#### Hotel Operations

##### `GetHotelByID(ctx context.Context, id string) (Hotel, error)`
**Description:** Retrieves a hotel by its unique identifier using cache-aside pattern.

**Flow:**
1. Check cache for hotel
2. If cache miss, fetch from MongoDB
3. Store in cache for future requests
4. Convert DAO → Domain and return

**Use Case:** Display hotel details on search results or detail page.

**Example:**
```go
hotel, err := service.GetHotelByID(ctx, "hotel-123")
if err != nil {
    // Handle not found
}
fmt.Printf("Hotel: %s, Price: $%.2f/night\n", hotel.Name, hotel.PricePerNight)
```

---

##### `Create(ctx context.Context, hotel Hotel) (string, error)`
**Description:** Creates a new hotel in the system.

**Flow:**
1. Convert Domain → DAO
2. Insert into MongoDB (generates ID)
3. Cache the new hotel
4. Publish `HotelNew` event to message queue
5. Return generated ID

**Use Case:** Admin adds a new hotel to the platform.

**Example:**
```go
newHotel := Hotel{
    Name:          "Grand Plaza Hotel",
    City:          "New York",
    PricePerNight: 299.99,
    AvaiableRooms: 50,
}
id, err := service.Create(ctx, newHotel)
```

---

##### `Update(ctx context.Context, hotel Hotel) error`
**Description:** Updates an existing hotel's information.

**Flow:**
1. Convert Domain → DAO
2. Update in MongoDB
3. Update cache
4. Return result

**Use Case:** Admin modifies hotel details (price change, room count update, etc.).

**Example:**
```go
hotel.PricePerNight = 349.99
hotel.AvaiableRooms = 45
err := service.Update(ctx, hotel)
```

---

##### `Delete(ctx context.Context, id string) error`
**Description:** Deletes a hotel and all associated reservations.

**Flow:**
1. Delete all reservations for this hotel (MongoDB)
2. Delete hotel from MongoDB
3. Delete from cache
4. Delete reservations from cache

**Use Case:** Remove a hotel that's permanently closed.

**Example:**
```go
err := service.Delete(ctx, "hotel-123")
```

---

#### Reservation Operations

##### `CreateReservation(ctx context.Context, reservation Reservation) (string, error)`
**Description:** Creates a new reservation for a hotel.

**Flow:**
1. Convert Domain → DAO
2. Insert into MongoDB
3. Cache the reservation
4. Return generated reservation ID

**Use Case:** User books a hotel room.

**Example:**
```go
reservation := Reservation{
    HotelID:  "hotel-123",
    UserID:   "user-456",
    CheckIn:  time.Parse("2006-01-02", "2025-12-01"),
    CheckOut: time.Parse("2006-01-02", "2025-12-05"),
}
resID, err := service.CreateReservation(ctx, reservation)
```

---

##### `GetReservationByID(ctx context.Context, id string) (Reservation, error)`
**Description:** Retrieves a reservation by ID using cache-aside pattern.

**Flow:**
1. Check cache
2. If cache miss, fetch from MongoDB
3. Cache the reservation
4. Convert DAO → Domain and return

**Use Case:** Display reservation details or confirmation page.

---

##### `CancelReservation(ctx context.Context, id string) error`
**Description:** Cancels an existing reservation.

**Flow:**
1. Delete from MongoDB
2. Remove from cache

**Use Case:** User cancels their booking.

**Example:**
```go
err := service.CancelReservation(ctx, "reservation-789")
```

---

##### `GetReservationsByHotelID(ctx context.Context, hotelID string) ([]Reservation, error)`
**Description:** Retrieves all reservations for a specific hotel.

**Use Case:** Admin views all bookings for a hotel, or calculate occupancy.

**Example:**
```go
reservations, err := service.GetReservationsByHotelID(ctx, "hotel-123")
fmt.Printf("Total reservations: %d\n", len(reservations))
```

---

##### `GetReservationsByUserID(ctx context.Context, userID string) ([]Reservation, error)`
**Description:** Retrieves all reservations made by a specific user.

**Use Case:** User views their booking history.

**Example:**
```go
myReservations, err := service.GetReservationsByUserID(ctx, "user-456")
for _, res := range myReservations {
    fmt.Printf("Hotel: %s, Check-in: %s\n", res.HotelName, res.CheckIn)
}
```

---

##### `GetReservationsByUserAndHotelID(ctx context.Context, hotelID, userID string) ([]Reservation, error)`
**Description:** Retrieves reservations for a specific user at a specific hotel.

**Use Case:** Check if user has existing reservations at this hotel (prevent double-booking).

**Example:**
```go
existing, err := service.GetReservationsByUserAndHotelID(ctx, "hotel-123", "user-456")
if len(existing) > 0 {
    fmt.Println("User already has a reservation at this hotel")
}
```

---

##### `GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error)`
**Description:** Checks availability for multiple hotels concurrently for a date range.

**Algorithm:**
1. For each hotel (using goroutines):
   - Get hotel details (available rooms count)
   - Count active reservations overlapping with requested dates
   - Available = (AvaiableRooms - ActiveReservations) > 0
2. Return map[hotelID]bool indicating availability

**Use Case:** Search results page showing which hotels have availability.

**Example:**
```go
hotelIDs := []string{"hotel-123", "hotel-456", "hotel-789"}
availability, err := service.GetAvailability(ctx, hotelIDs, "2025-12-01", "2025-12-05")

for hotelID, isAvailable := range availability {
    if isAvailable {
        fmt.Printf("Hotel %s: AVAILABLE\n", hotelID)
    } else {
        fmt.Printf("Hotel %s: FULLY BOOKED\n", hotelID)
    }
}
```

---

## 🗄️ Repository Layer

### Interface Definition
All repository implementations (Mongo, Cache, Mock) implement this interface:

```go
type Repository interface {
    // Hotel CRUD
    GetHotelByID(ctx context.Context, id string) (Hotel, error)
    Create(ctx context.Context, hotel Hotel) (string, error)
    Update(ctx context.Context, hotel Hotel) error
    Delete(ctx context.Context, id string) error
    
    // Reservation CRUD
    CreateReservation(ctx context.Context, reservation Reservation) (string, error)
    GetReservationByID(ctx context.Context, id string) (Reservation, error)
    CancelReservation(ctx context.Context, id string) error
    GetReservationsByHotelID(ctx context.Context, hotelID string) ([]Reservation, error)
    GetReservationsByUserAndHotelID(ctx context.Context, hotelID, userID string) ([]Reservation, error)
    GetReservationsByUserID(ctx context.Context, userID string) ([]Reservation, error)
    DeleteReservationsByHotelID(ctx context.Context, hotelID string) error
    
    // Availability
    GetAvailability(ctx context.Context, hotelIDs []string, checkIn, checkOut string) (map[string]bool, error)
}
```

### Implementations

#### 1. MongoDB Repository (`hotels_mongo.go`)
**Purpose:** Primary data store with persistence.

**Key Features:**
- ObjectID generation for unique identifiers
- Indexed queries for performance (hotelID, userID)
- Atomic operations for consistency
- Concurrent availability checking using goroutines

**Connection String:**
```
mongodb://[username:password@]host:port/database
```

#### 2. Cache Repository (`hotels_cache.go`)
**Purpose:** In-memory caching layer for fast reads.

**Implementation:** `karlseguin/ccache` (LRU cache)

**Key Features:**
- Configurable TTL (time-to-live)
- LRU eviction policy
- Thread-safe operations
- Max size limits to prevent memory exhaustion

**Configuration:**
```go
CacheConfig{
    MaxSize:       1000,        // Max items
    ItemsToPrune:  100,         // Items to remove when full
    Duration:      30 * time.Second, // TTL
}
```

#### 3. Mock Repository (`hotels_mock.go`)
**Purpose:** In-memory implementation for testing.

**Key Features:**
- No external dependencies
- Deterministic behavior
- Fast execution
- Easy data setup for tests

**Usage:**
```go
mainRepo := hotels.NewMock()
cacheRepo := hotels.NewMockCache()
events := queues.NewMock()
service := services.NewService(mainRepo, cacheRepo, &events)
// Run tests...
```

---

## 🧪 Testing Strategy

### Test Types

#### 1. Controller Tests (`internal/controllers/**`)
**What:** Lightweight API integration tests (Gin + `httptest`) with a mocked `Service`.

**Coverage:**
- ✅ Routing (paths + params)
- ✅ JSON binding (400 on invalid payloads)
- ✅ AuthN/AuthZ (401/403 via JWT middleware + roles)
- ✅ Response status codes

**Run:**
```bash
go test ./internal/controllers/... -v
```

#### 2. Service Tests (`internal/services/**`)
**What:** Unit tests for business logic using mock repositories (main + cache) and a mock queue.

**Coverage:**
- ✅ Hotel CRUD operations
- ✅ Reservation lifecycle
- ✅ Cache-aside behavior (including cache population)
- ✅ Availability date logic (checkout is excluded)

**Run:**
```bash
go test ./internal/services -v
```

#### All tests
```bash
go test ./... -v
```

---

## 🚀 Usage Examples

### Complete Workflow Example

```go
// 1. Initialize repositories
mongoRepo := hotels.NewMongo(MongoConfig{
    Host:       "localhost",
    Port:       "27017",
    Database:   "hotels_db",
    Collection_hotels: "hotels",
    Collection_reservations: "reservations",
})

cacheRepo := hotels.NewCache(CacheConfig{
    MaxSize:  1000,
    Duration: 30 * time.Second,
})

// 2. Create service
// NOTE: the service publishes hotel events; provide a Queue implementation (RabbitMQ or mock).
events := queues.NewMock()
hotelService := services.NewService(mongoRepo, cacheRepo, &events)

// 3. Create a new hotel
newHotel := domain.Hotel{
    Name:          "Sunset Resort",
    City:          "Miami",
    Country:       "USA",
    Rating:        4.5,
    PricePerNight: 199.99,
    AvaiableRooms: 30,
    Amenities:     []string{"WiFi", "Pool", "Spa", "Restaurant"},
}

hotelID, err := hotelService.Create(context.Background(), newHotel)
if err != nil {
    log.Fatal(err)
}

// 4. Make a reservation
reservation := domain.Reservation{
    HotelID:   hotelID,
    HotelName: "Sunset Resort",
    UserID:    "user-123",
    CheckIn:   time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
    CheckOut:  time.Date(2025, 12, 5, 0, 0, 0, 0, time.UTC),
}

resID, err := hotelService.CreateReservation(context.Background(), reservation)

// 5. Check availability for multiple hotels
hotelIDs := []string{hotelID, "other-hotel-id"}
availability, err := hotelService.GetAvailability(
    context.Background(),
    hotelIDs,
    "2025-12-01",
    "2025-12-05",
)

for id, available := range availability {
    if available {
        fmt.Printf("Hotel %s is available!\n", id)
    }
}

// 6. Get user's reservations
myReservations, err := hotelService.GetReservationsByUserID(context.Background(), "user-123")

// 7. Cancel a reservation
err = hotelService.CancelReservation(context.Background(), resID)
```

---

## 🔐 Error Handling

### HTTP error responses

Controllers return JSON with an `error` field. Typical status codes:

- **400 Bad Request**: invalid JSON/body
- **401 Unauthorized**: missing/invalid `Authorization: Bearer <token>`
- **403 Forbidden**: role/user mismatch (e.g. non-admin calling `/admin/*`, user creating/canceling a reservation for another user)
- **404 Not Found**: hotel/reservation not found
- **500 Internal Server Error**: unexpected service/repository failure

---

## 📊 Performance Considerations

### Cache Strategy
- Cache-aside pattern for hotel/reservation reads
- TTL is configurable (default: 30s)

### Concurrent Operations
- Availability checking uses goroutines (1 per hotel)
- Parallel checks improve throughput when querying multiple hotels

### Database Indexes
**Recommended MongoDB indexes:**
```javascript
// Hotels collection
db.hotels.createIndex({ "_id": 1 })

// Reservations collection
db.reservations.createIndex({ "hotel_id": 1 })
db.reservations.createIndex({ "user_id": 1 })
db.reservations.createIndex({ "hotel_id": 1, "user_id": 1 })
db.reservations.createIndex({ "check_in": 1, "check_out": 1 })
```

---

## 🛠️ Future Enhancements

### Planned Features
- [ ] Redis cache implementation (distributed caching)
- [ ] GraphQL API layer
- [ ] Real-time availability updates via WebSockets
- [ ] Hotel search with filters (price range, rating, amenities)
- [ ] Pagination for large result sets
- [x] JWT authentication + role-based access (admin vs logged user)
- [ ] Rate limiting
- [ ] Metrics and monitoring (Prometheus)

### Event-Driven Architecture
- [ ] Publish events: `HotelCreated`, `HotelUpdated`, `ReservationMade`, `ReservationCancelled`
- [ ] Consumer services: Search indexing, analytics, notifications

---

## 📚 Dependencies

```go
// Core
github.com/gin-gonic/gin         // HTTP framework
github.com/gin-contrib/cors      // CORS middleware
github.com/golang-jwt/jwt/v5     // JWT validation
go.mongodb.org/mongo-driver      // MongoDB driver
github.com/karlseguin/ccache     // In-memory cache
github.com/streadway/amqp        // RabbitMQ client
github.com/google/uuid           // UUID generation (mocks)
```

---

## 🎓 Learning Outcomes

This project demonstrates:
- ✅ **Clean Architecture:** Separation of concerns (DAO, Domain, Service, Repository)
- ✅ **Design Patterns:** Repository pattern, Cache-aside, Dependency injection
- ✅ **Go Best Practices:** Interfaces, context handling, error wrapping
- ✅ **Concurrency:** Goroutines for parallel operations
- ✅ **Testing:** Unit tests with mocks, integration tests
- ✅ **Microservices:** Service isolation, event-driven communication
- ✅ **Database Design:** NoSQL modeling, indexing strategies

---

## 📞 Contact

**Developer:** Julian Irusta Roure
**Repository:** [GitHub Link - Add your repo URL]  
**Portfolio:** [Your Portfolio URL]

---

**Last Updated:** January 2026