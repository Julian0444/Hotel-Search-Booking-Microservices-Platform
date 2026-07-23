# StayLux - Hotel Booking Frontend

Modern React frontend for the StayLux hotel search and booking platform.

## 🚀 Tech Stack

- **React 19** - UI Library
- **Vite** - Build tool and dev server
- **Material UI v7** - Component library
- **React Router v6** - Navigation
- **Axios** - HTTP client
- **React Hook Form** - Form handling
- **date-fns** - Date utilities

## 📁 Project Structure

```
frontend/src/
├── components/           # Reusable components
│   ├── common/          # Shared UI components
│   ├── hotels/          # Hotel-specific components
│   └── layout/          # Layout components (Navbar, Footer)
├── constants/           # Application constants
│   └── index.js         # Routes, API config, etc.
├── context/             # React contexts
│   └── AuthContext.jsx  # Authentication context
├── hooks/               # Custom hooks
│   └── useAuth.js       # Auth hook
├── pages/               # Page components
│   ├── admin/           # Admin pages
│   ├── Home.jsx
│   ├── Search.jsx
│   ├── HotelDetail.jsx
│   ├── Login.jsx
│   ├── Register.jsx
│   └── MyReservations.jsx
├── services/            # API services
│   ├── api.js           # Base axios config
│   ├── auth.service.js  # Auth endpoints
│   ├── hotels.service.js# Hotels endpoints
│   ├── reservations.service.js
│   └── admin.service.js # Admin endpoints
├── types/               # JSDoc type definitions
│   └── index.js         # Data types
├── utils/               # Utility functions
│   ├── helpers.js       # Helper functions
│   └── validators.js    # Validation utilities
├── theme/               # MUI theme config
│   └── theme.js
├── App.jsx              # Main component
├── main.jsx             # Entry point
└── index.css            # Global styles
```

## 🛠️ Installation

### Local Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev
```

The application will be available at `http://localhost:5173`

### With Docker

```bash
# From project root
docker-compose up -d frontend
```

The application will be available at `http://localhost:5173`

## 🔧 Configuration

### Environment Variables

Create a `.env` file in the frontend root:

```env
VITE_API_URL=http://localhost/api/v1
```

- `VITE_API_URL`: API Gateway URL (nginx)

## 📱 Pages

### Public
- `/` - Home page with search and featured hotels
- `/search` - Hotel search with filters
- `/hotels/:id` - Hotel details with booking
- `/login` - User login
- `/register` - User registration

### Protected (requires authentication)
- `/reservations` - My reservations

### Admin (administrators only)
- `/admin` - Admin dashboard
- `/admin/hotels/new` - Create new hotel
- `/admin/hotels/:id/edit` - Edit existing hotel

## 🎨 Design

The frontend features an elegant design inspired by luxury hotels:

- **Color Palette**: Deep blue (#1a365d) with golden accents (#c6a961)
- **Typography**: Cormorant Garamond (headings) + Source Sans 3 (body)
- **Animations**: Smooth transitions and hover effects
- **Responsive**: Mobile-first design

## 🔌 API Endpoints

The frontend connects to the API Gateway (nginx) which routes to microservices:

| Endpoint | Service | Description |
|----------|---------|-------------|
| `/login` | users-api | Authentication |
| `/users` | users-api | User management |
| `/search` | search-api | Hotel search |
| `/hotels` | hotels-api | Hotel information |
| `/reservations` | hotels-api | Reservations |
| `/admin/*` | hotels-api | Administration |

## 📝 Available Scripts

```bash
npm run dev      # Development server
npm run build    # Production build
npm run preview  # Preview production build
npm run lint     # Run linter
```

## 🏗️ Architecture Decisions

### Service Layer
API calls are organized by domain (auth, hotels, reservations, admin) for better maintainability and Single Responsibility Principle.

### Type Definitions
JSDoc type definitions in `/types` provide IDE autocompletion and serve as documentation, making future TypeScript migration easier.

### Constants
Centralized constants prevent magic strings and make configuration changes easier.

### Custom Hooks
Authentication logic is encapsulated in `useAuth` hook for reusability across components.

## 🐳 Docker

The Dockerfile includes:
1. **Build stage**: Compiles the application with Node.js
2. **Production stage**: Serves with optimized nginx

```bash
# Manual build
docker build -t staylux-frontend ./frontend

# Run
docker run -p 5173:80 staylux-frontend
```
