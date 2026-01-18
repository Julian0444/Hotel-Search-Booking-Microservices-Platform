// MongoDB Seed Script for Hotels
// Run this inside the MongoDB container:
// 1. docker exec -it hotels-mongo mongosh -u root -p root
// 2. use hotels-api
// 3. Copy and paste the commands below

// Switch to the hotels database
// use hotels-api

// Clear existing hotels (optional)
// db.hotels.deleteMany({})

// Insert sample hotels
db.hotels.insertMany([
  {
    name: "The Grand Palace Hotel",
    description: "Experience luxury at its finest in the heart of Manhattan. This 5-star hotel features breathtaking views of Central Park, world-class dining, and impeccable service that defines New York elegance.",
    address: "768 5th Avenue",
    city: "New York",
    state: "NY",
    country: "USA",
    phone: "+1-212-555-0100",
    email: "reservations@grandpalace.com",
    price_per_night: 599,
    rating: 4.8,
    avaiable_rooms: 45,
    check_in_time: new Date("2024-01-01T15:00:00Z"),
    check_out_time: new Date("2024-01-01T11:00:00Z"),
    amenities: ["wifi", "pool", "spa", "gym", "restaurant", "bar", "room_service", "parking", "concierge"],
    images: [
      "https://images.unsplash.com/photo-1566073771259-6a8506099945?w=800",
      "https://images.unsplash.com/photo-1582719508461-905c673771fd?w=800",
      "https://images.unsplash.com/photo-1520250497591-112f2f40a3f4?w=800"
    ]
  },
  {
    name: "Oceanview Resort & Spa",
    description: "A tropical paradise on the stunning beaches of Miami. Enjoy pristine white sand, crystal-clear waters, and rejuvenating spa treatments in this beachfront sanctuary.",
    address: "1901 Collins Avenue",
    city: "Miami Beach",
    state: "FL",
    country: "USA",
    phone: "+1-305-555-0200",
    email: "info@oceanviewresort.com",
    price_per_night: 449,
    rating: 4.6,
    avaiable_rooms: 78,
    check_in_time: new Date("2024-01-01T16:00:00Z"),
    check_out_time: new Date("2024-01-01T10:00:00Z"),
    amenities: ["wifi", "pool", "spa", "gym", "restaurant", "bar", "beach_access", "water_sports"],
    images: [
      "https://images.unsplash.com/photo-1571896349842-33c89424de2d?w=800",
      "https://images.unsplash.com/photo-1551882547-ff40c63fe5fa?w=800",
      "https://images.unsplash.com/photo-1542314831-068cd1dbfeeb?w=800"
    ]
  },
  {
    name: "Mountain Lodge Retreat",
    description: "Escape to the majestic Rocky Mountains. This cozy lodge offers stunning alpine views, world-class skiing, and a warm fireplace to come home to after your adventures.",
    address: "500 Ski Run Boulevard",
    city: "Aspen",
    state: "CO",
    country: "USA",
    phone: "+1-970-555-0300",
    email: "stay@mountainlodge.com",
    price_per_night: 379,
    rating: 4.7,
    avaiable_rooms: 32,
    check_in_time: new Date("2024-01-01T15:00:00Z"),
    check_out_time: new Date("2024-01-01T11:00:00Z"),
    amenities: ["wifi", "spa", "gym", "restaurant", "bar", "ski_storage", "fireplace", "parking"],
    images: [
      "https://images.unsplash.com/photo-1520250497591-112f2f40a3f4?w=800",
      "https://images.unsplash.com/photo-1596394516093-501ba68a0ba6?w=800",
      "https://images.unsplash.com/photo-1445019980597-93fa8acb246c?w=800"
    ]
  },
  {
    name: "Historic Plaza Inn",
    description: "Step into history at this beautifully restored 19th-century hotel in the heart of San Francisco. Victorian elegance meets modern comfort with cable car access at your doorstep.",
    address: "335 Powell Street",
    city: "San Francisco",
    state: "CA",
    country: "USA",
    phone: "+1-415-555-0400",
    email: "hello@historicplazainn.com",
    price_per_night: 299,
    rating: 4.5,
    avaiable_rooms: 56,
    check_in_time: new Date("2024-01-01T15:00:00Z"),
    check_out_time: new Date("2024-01-01T12:00:00Z"),
    amenities: ["wifi", "restaurant", "bar", "room_service", "concierge", "laundry"],
    images: [
      "https://images.unsplash.com/photo-1564501049412-61c2a3083791?w=800",
      "https://images.unsplash.com/photo-1566073771259-6a8506099945?w=800",
      "https://images.unsplash.com/photo-1582719508461-905c673771fd?w=800"
    ]
  },
  {
    name: "Desert Oasis Hotel",
    description: "Discover serenity in the Sonoran Desert. This boutique hotel offers stunning sunset views, a world-class spa, and the tranquility of the Arizona wilderness.",
    address: "7575 E Princess Drive",
    city: "Scottsdale",
    state: "AZ",
    country: "USA",
    phone: "+1-480-555-0500",
    email: "reservations@desertoasis.com",
    price_per_night: 329,
    rating: 4.4,
    avaiable_rooms: 41,
    check_in_time: new Date("2024-01-01T16:00:00Z"),
    check_out_time: new Date("2024-01-01T11:00:00Z"),
    amenities: ["wifi", "pool", "spa", "gym", "restaurant", "golf", "parking"],
    images: [
      "https://images.unsplash.com/photo-1551882547-ff40c63fe5fa?w=800",
      "https://images.unsplash.com/photo-1571003123894-1f0594d2b5d9?w=800",
      "https://images.unsplash.com/photo-1520250497591-112f2f40a3f4?w=800"
    ]
  },
  {
    name: "Riverside Boutique Hotel",
    description: "A charming boutique experience along the Chicago River. Modern design, rooftop dining, and easy access to the city's best attractions make this the perfect urban escape.",
    address: "85 E Wacker Drive",
    city: "Chicago",
    state: "IL",
    country: "USA",
    phone: "+1-312-555-0600",
    email: "info@riversideboutique.com",
    price_per_night: 279,
    rating: 4.3,
    avaiable_rooms: 63,
    check_in_time: new Date("2024-01-01T15:00:00Z"),
    check_out_time: new Date("2024-01-01T11:00:00Z"),
    amenities: ["wifi", "gym", "restaurant", "bar", "rooftop", "room_service"],
    images: [
      "https://images.unsplash.com/photo-1542314831-068cd1dbfeeb?w=800",
      "https://images.unsplash.com/photo-1564501049412-61c2a3083791?w=800",
      "https://images.unsplash.com/photo-1571896349842-33c89424de2d?w=800"
    ]
  },
  {
    name: "Vineyard Estate Resort",
    description: "Immerse yourself in California wine country. This elegant resort offers wine tastings, gourmet dining, and breathtaking views of rolling vineyards in Napa Valley.",
    address: "1200 Rutherford Road",
    city: "Napa",
    state: "CA",
    country: "USA",
    phone: "+1-707-555-0700",
    email: "stay@vineyardestate.com",
    price_per_night: 489,
    rating: 4.9,
    avaiable_rooms: 28,
    check_in_time: new Date("2024-01-01T15:00:00Z"),
    check_out_time: new Date("2024-01-01T11:00:00Z"),
    amenities: ["wifi", "pool", "spa", "restaurant", "bar", "wine_tasting", "parking", "concierge"],
    images: [
      "https://images.unsplash.com/photo-1596394516093-501ba68a0ba6?w=800",
      "https://images.unsplash.com/photo-1445019980597-93fa8acb246c?w=800",
      "https://images.unsplash.com/photo-1566073771259-6a8506099945?w=800"
    ]
  },
  {
    name: "Lakefront Lodge",
    description: "Peaceful retreat on the shores of Lake Tahoe. Perfect for outdoor enthusiasts seeking hiking, kayaking, and stunning mountain lake views year-round.",
    address: "300 Lakeshore Boulevard",
    city: "South Lake Tahoe",
    state: "CA",
    country: "USA",
    phone: "+1-530-555-0800",
    email: "reservations@lakefrontlodge.com",
    price_per_night: 349,
    rating: 4.6,
    avaiable_rooms: 37,
    check_in_time: new Date("2024-01-01T16:00:00Z"),
    check_out_time: new Date("2024-01-01T10:00:00Z"),
    amenities: ["wifi", "spa", "restaurant", "bar", "kayak_rental", "fireplace", "parking"],
    images: [
      "https://images.unsplash.com/photo-1571003123894-1f0594d2b5d9?w=800",
      "https://images.unsplash.com/photo-1520250497591-112f2f40a3f4?w=800",
      "https://images.unsplash.com/photo-1582719508461-905c673771fd?w=800"
    ]
  },
  {
    name: "Urban Loft Hotel",
    description: "Industrial chic meets comfort in downtown Seattle. This trendy hotel features exposed brick, artisan coffee, and easy access to Pike Place Market.",
    address: "1415 5th Avenue",
    city: "Seattle",
    state: "WA",
    country: "USA",
    phone: "+1-206-555-0900",
    email: "hello@urbanlofthotel.com",
    price_per_night: 249,
    rating: 4.2,
    avaiable_rooms: 52,
    check_in_time: new Date("2024-01-01T15:00:00Z"),
    check_out_time: new Date("2024-01-01T11:00:00Z"),
    amenities: ["wifi", "gym", "restaurant", "coffee_bar", "coworking", "bike_rental"],
    images: [
      "https://images.unsplash.com/photo-1551882547-ff40c63fe5fa?w=800",
      "https://images.unsplash.com/photo-1564501049412-61c2a3083791?w=800",
      "https://images.unsplash.com/photo-1542314831-068cd1dbfeeb?w=800"
    ]
  },
  {
    name: "Beachside Paradise Hotel",
    description: "Your Hawaiian dream awaits. This oceanfront property offers traditional Hawaiian hospitality, luau experiences, and direct beach access on the beautiful island of Maui.",
    address: "2365 Kaanapali Parkway",
    city: "Lahaina",
    state: "HI",
    country: "USA",
    phone: "+1-808-555-1000",
    email: "aloha@beachsideparadise.com",
    price_per_night: 549,
    rating: 4.8,
    avaiable_rooms: 89,
    check_in_time: new Date("2024-01-01T15:00:00Z"),
    check_out_time: new Date("2024-01-01T11:00:00Z"),
    amenities: ["wifi", "pool", "spa", "gym", "restaurant", "bar", "beach_access", "water_sports", "luau"],
    images: [
      "https://images.unsplash.com/photo-1571896349842-33c89424de2d?w=800",
      "https://images.unsplash.com/photo-1520250497591-112f2f40a3f4?w=800",
      "https://images.unsplash.com/photo-1566073771259-6a8506099945?w=800"
    ]
  },
  {
    name: "Southern Charm Inn",
    description: "Experience true Southern hospitality in the heart of Charleston. This elegant inn offers antebellum architecture, garden courtyards, and acclaimed Lowcountry cuisine.",
    address: "225 Meeting Street",
    city: "Charleston",
    state: "SC",
    country: "USA",
    phone: "+1-843-555-1100",
    email: "stay@southerncharminn.com",
    price_per_night: 319,
    rating: 4.7,
    avaiable_rooms: 34,
    check_in_time: new Date("2024-01-01T15:00:00Z"),
    check_out_time: new Date("2024-01-01T11:00:00Z"),
    amenities: ["wifi", "restaurant", "bar", "garden", "room_service", "concierge", "parking"],
    images: [
      "https://images.unsplash.com/photo-1564501049412-61c2a3083791?w=800",
      "https://images.unsplash.com/photo-1596394516093-501ba68a0ba6?w=800",
      "https://images.unsplash.com/photo-1445019980597-93fa8acb246c?w=800"
    ]
  },
  {
    name: "Downtown Executive Suites",
    description: "The perfect blend of business and leisure in the nation's capital. Steps from the National Mall, this hotel offers executive amenities and stunning monument views.",
    address: "900 F Street NW",
    city: "Washington",
    state: "DC",
    country: "USA",
    phone: "+1-202-555-1200",
    email: "reservations@executivesuites.com",
    price_per_night: 389,
    rating: 4.4,
    avaiable_rooms: 67,
    check_in_time: new Date("2024-01-01T15:00:00Z"),
    check_out_time: new Date("2024-01-01T12:00:00Z"),
    amenities: ["wifi", "gym", "restaurant", "bar", "business_center", "room_service", "parking"],
    images: [
      "https://images.unsplash.com/photo-1542314831-068cd1dbfeeb?w=800",
      "https://images.unsplash.com/photo-1571003123894-1f0594d2b5d9?w=800",
      "https://images.unsplash.com/photo-1551882547-ff40c63fe5fa?w=800"
    ]
  }
]);

// Verify insertion
print("Hotels inserted successfully!");
print("Total hotels: " + db.hotels.countDocuments());

// Show all hotels
db.hotels.find({}, { name: 1, city: 1, price_per_night: 1, rating: 1 }).pretty();
