# StayLux - Hotel Booking Frontend

Modern React frontend for the StayLux hotel search and booking platform.

## 🚀 Tech Stack

- **React 19** - UI Library
- **Vite 7** - Build tool and dev server
- **Material UI v7** - Component library
- **React Router 8** - Navigation
- **TanStack Query 5** - Server state (caching, retries, invalidation)
- **Axios** - HTTP client
- **React Hook Form** - Form handling
- **Vitest + MSW + Playwright** - Testing (unit/component + E2E)

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
│   ├── queries/         # TanStack Query hooks (search, hotel, reservations, admin)
│   ├── mutations/       # TanStack Query mutations (booking, cancel, admin CRUD)
│   ├── useAuth.js       # Auth hook
│   ├── useBooking.js    # Booking form state (shared desktop/mobile)
│   ├── useHotelSearchParams.js  # URL as source of truth for search
│   └── useUnsavedChanges.js     # Unsaved-changes guard for forms
├── pages/               # Page components
│   ├── Admin/           # Admin pages (Dashboard, HotelForm)
│   ├── Home.jsx
│   ├── Search.jsx
│   ├── HotelDetail.jsx
│   ├── Login.jsx
│   ├── Register.jsx
│   ├── MyReservations.jsx
│   └── NotFound.jsx
├── services/            # API services
│   ├── api.js           # Base axios config
│   ├── auth.service.js  # Auth endpoints
│   ├── hotels.service.js# Hotels endpoints
│   ├── reservations.service.js
│   └── admin.service.js # Admin endpoints
├── types/               # JSDoc type definitions
│   └── index.js         # Data types
├── utils/               # Utility functions
│   ├── dateOnly.js      # Civil-date helpers (DST-safe, no timezones)
│   ├── money.js         # Integer-cents money formatting
│   ├── idempotency.js   # Per-attempt Idempotency-Key tracker
│   ├── reservations.js  # Reservation grouping/status helpers
│   ├── prefetch.js      # Route-chunk prefetch on hover/focus
│   ├── helpers.js       # Helper functions
│   └── validators.js    # Validation utilities
├── test/                # Test infra (MSW server + handlers, fixtures, render helpers)
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
# From project root (the frontend sits behind a Compose profile)
docker compose --profile frontend up -d --build
```

The application will be available at `http://localhost:5173`

## 🔧 Configuration

### Environment Variables

The default API URL is `/api/v1` on the SPA origin. No frontend `.env` is required.

- Development: Vite proxies `/api` to the local TLS gateway at `https://localhost`, accepting its local self-signed certificate on the server side.
- Built SPA: its nginx proxies `/api` to `https://nginx:443` on the Compose network. Browser requests stay on `http://localhost:5173`, avoiding cross-origin preflights and certificate prompts.
- `VITE_API_URL` remains an optional build-time override for a deliberately configured gateway. A different browser origin requires valid TLS trust and matching CORS configuration.

The root `.env` and local gateway certificates must exist; follow the [main quickstart](../README.md).

## 📱 Pages

### Public
- `/` - Home page with search and featured hotels
- `/search` - Hotel search by name, city or country, with sorting and pagination
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

All endpoints below are relative to `/api/v1`. The API gateway routes them to microservices:

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
npm run dev            # Development server
npm run build          # Production build
npm run preview        # Preview production build
npm run lint           # Run linter
npm run test           # Vitest in watch mode
npm run test:run       # Unit/component suite (Testing Library + MSW)
npm run test:coverage  # Suite with coverage thresholds
npm run test:e2e       # Playwright E2E (needs the Docker stack with --profile frontend)
npm run check          # lint + coverage + build (CI gate)
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

# Run with its gateway on the Compose network
# (the frontend nginx requires the `nginx` service alias)
docker compose --profile frontend up -d --build
```

## Booking and administration

- Login and registration preserve the selected hotel, dates, rooms and guests in router navigation state. Idempotency keys belong to an attempt and are reset when the user changes.
- Reservations load in pages of 20 through **Load more bookings**, including cancellations. Tabs and counts describe the bookings loaded so far.
- Booking and cancellation invalidate availability queries. Hotel detail loads inventory from the real API.
- Hotel editing sends a complete PUT. Capacity, price and rating allow zero; required name/address/city/country and times cannot be empty. Optional text and lists can be cleared. The backend rejects unsafe capacity reductions and deleting a hotel with reservation history.
- The data router protects dirty forms on internal links, Sign out and browser Back. Reload/close uses the browser's native warning. Successful saves clear the warning.
- E2E uses the built SPA and real APIs: normal login, registration, destination search, availability, reservation/cancellation, admin editing, pagination and dirty navigation. Search checks use ordinary URLs; no cache-busters or blanket CORS/network exception filters.
