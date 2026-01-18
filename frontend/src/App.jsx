/**
 * Main Application Component
 * Sets up routing, theme, and authentication context
 */

import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider, CssBaseline } from '@mui/material';
import { AuthProvider, useAuth } from './context/AuthContext';
import theme from './theme/theme';
import { ROUTES } from './constants';

// Layout
import { Layout } from './components/Layout';

// Pages
import Home from './pages/Home';
import Search from './pages/Search';
import Login from './pages/Login';
import Register from './pages/Register';
import HotelDetail from './pages/HotelDetail';
import MyReservations from './pages/MyReservations';
import { Dashboard, HotelForm } from './pages/Admin';

/**
 * Protected Route Component
 * Restricts access based on authentication and admin status
 */
const ProtectedRoute = ({ children, adminOnly = false }) => {
  const { isAuthenticated, isAdmin, loading } = useAuth();

  if (loading) {
    return null;
  }

  if (!isAuthenticated) {
    return <Navigate to={ROUTES.LOGIN} replace />;
  }

  if (adminOnly && !isAdmin) {
    return <Navigate to={ROUTES.HOME} replace />;
  }

  return children;
};

/**
 * App content wrapped in AuthProvider
 */
const AppContent = () => {
  return (
    <Router>
      <Routes>
        {/* Public routes with layout */}
        <Route
          path={ROUTES.HOME}
          element={
            <Layout>
              <Home />
            </Layout>
          }
        />
        <Route
          path={ROUTES.SEARCH}
          element={
            <Layout>
              <Search />
            </Layout>
          }
        />
        <Route
          path="/hotels/:id"
          element={
            <Layout>
              <HotelDetail />
            </Layout>
          }
        />

        {/* Auth routes - no layout */}
        <Route path={ROUTES.LOGIN} element={<Login />} />
        <Route path={ROUTES.REGISTER} element={<Register />} />

        {/* Protected routes */}
        <Route
          path={ROUTES.RESERVATIONS}
          element={
            <ProtectedRoute>
              <Layout>
                <MyReservations />
              </Layout>
            </ProtectedRoute>
          }
        />

        {/* Admin routes */}
        <Route
          path={ROUTES.ADMIN}
          element={
            <ProtectedRoute adminOnly>
              <Layout>
                <Dashboard />
              </Layout>
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin/hotels/new"
          element={
            <ProtectedRoute adminOnly>
              <HotelForm />
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin/hotels/:id/edit"
          element={
            <ProtectedRoute adminOnly>
              <HotelForm />
            </ProtectedRoute>
          }
        />

        {/* 404 redirect */}
        <Route path="*" element={<Navigate to={ROUTES.HOME} replace />} />
      </Routes>
    </Router>
  );
};

/**
 * Root App Component
 */
function App() {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
