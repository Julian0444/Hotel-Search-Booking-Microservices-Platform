/**
 * Admin Dashboard Page
 * Administrative panel for managing hotels and users
 */

import { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import {
  Box,
  Container,
  Typography,
  Grid,
  Card,
  CardContent,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  IconButton,
  Chip,
  Skeleton,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Snackbar,
  Tabs,
  Tab,
  Tooltip,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Hotel as HotelIcon,
  Person as PersonIcon,
  AdminPanelSettings as AdminIcon,
  Refresh as RefreshIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { hotelsService, adminService, authService } from '../../services';
import { useAuth } from '../../context/AuthContext';
import { ROUTES, USER_ROLES } from '../../constants';
import { formatPrice } from '../../utils/helpers';

const Dashboard = () => {
  const navigate = useNavigate();
  const { isAdmin, isAuthenticated } = useAuth();

  const [tabValue, setTabValue] = useState(0);
  const [hotels, setHotels] = useState([]);
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [deleteDialog, setDeleteDialog] = useState({ open: false, type: '', item: null });
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' });

  useEffect(() => {
    if (!isAuthenticated || !isAdmin) {
      navigate(ROUTES.LOGIN);
      return;
    }

    fetchData();
  }, [isAuthenticated, isAdmin, navigate]);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);

      const [hotelsRes, usersRes] = await Promise.all([
        hotelsService.search('', 0, 100),
        authService.getAllUsers(),
      ]);

      setHotels(hotelsRes || []);
      setUsers(usersRes || []);
    } catch (err) {
      console.error('Error fetching data:', err);
      setError('Error loading data');
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteClick = (type, item) => {
    setDeleteDialog({ open: true, type, item });
  };

  const handleDeleteConfirm = async () => {
    const { type, item } = deleteDialog;

    try {
      setDeleteLoading(true);

      if (type === 'hotel') {
        await adminService.deleteHotel(item.id);
        setHotels(hotels.filter((h) => h.id !== item.id));
        setSnackbar({ open: true, message: 'Hotel deleted successfully', severity: 'success' });
      } else if (type === 'user') {
        await authService.deleteUser(item.id);
        setUsers(users.filter((u) => u.id !== item.id));
        setSnackbar({ open: true, message: 'User deleted successfully', severity: 'success' });
      }
    } catch (err) {
      console.error('Error deleting:', err);
      setSnackbar({
        open: true,
        message: err.response?.data?.error || 'Error deleting item',
        severity: 'error',
      });
    } finally {
      setDeleteLoading(false);
      setDeleteDialog({ open: false, type: '', item: null });
    }
  };

  const stats = [
    {
      label: 'Hotels',
      value: hotels.length,
      icon: <HotelIcon sx={{ fontSize: 40 }} />,
      color: 'primary.main',
    },
    {
      label: 'Users',
      value: users.length,
      icon: <PersonIcon sx={{ fontSize: 40 }} />,
      color: 'secondary.main',
    },
    {
      label: 'Administrators',
      value: users.filter((u) => u.tipo === USER_ROLES.ADMIN).length,
      icon: <AdminIcon sx={{ fontSize: 40 }} />,
      color: 'success.main',
    },
  ];

  if (!isAuthenticated || !isAdmin) {
    return null;
  }

  return (
    <Box sx={{ bgcolor: 'background.default', minHeight: '100vh' }}>
      {/* Header */}
      <Box
        sx={{
          bgcolor: 'primary.main',
          py: 6,
          position: 'relative',
          overflow: 'hidden',
        }}
      >
        <Box
          sx={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            opacity: 0.1,
            backgroundImage: `url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='1'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E")`,
          }}
        />
        <Container maxWidth="lg" sx={{ position: 'relative', zIndex: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
              <AdminIcon sx={{ fontSize: 40, color: 'secondary.main', mr: 2 }} />
              <Box>
                <Typography variant="h3" sx={{ color: 'white', fontWeight: 600 }}>
                  Admin Dashboard
                </Typography>
                <Typography variant="body1" sx={{ color: 'rgba(255,255,255,0.7)' }}>
                  Manage hotels and users
                </Typography>
              </Box>
            </Box>
            <Button
              startIcon={<RefreshIcon />}
              onClick={fetchData}
              sx={{ color: 'white', borderColor: 'rgba(255,255,255,0.5)' }}
              variant="outlined"
            >
              Refresh
            </Button>
          </Box>
        </Container>
      </Box>

      <Container maxWidth="lg" sx={{ py: 4 }}>
        {error && (
          <Alert severity="error" sx={{ mb: 4 }}>
            {error}
          </Alert>
        )}

        {/* Stats Cards */}
        <Grid container spacing={3} sx={{ mb: 4 }}>
          {stats.map((stat, index) => (
            <Grid key={index} size={{ xs: 12, sm: 4 }}>
              <Card>
                <CardContent sx={{ p: 3 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <Box>
                      <Typography variant="overline" color="text.secondary">
                        {stat.label}
                      </Typography>
                      <Typography variant="h3" sx={{ fontWeight: 700, color: stat.color }}>
                        {loading ? <Skeleton width={60} /> : stat.value}
                      </Typography>
                    </Box>
                    <Box
                      sx={{
                        width: 64,
                        height: 64,
                        borderRadius: 2,
                        bgcolor: `${stat.color}15`,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        color: stat.color,
                      }}
                    >
                      {stat.icon}
                    </Box>
                  </Box>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>

        {/* Tabs */}
        <Card>
          <Tabs
            value={tabValue}
            onChange={(e, v) => setTabValue(v)}
            sx={{ borderBottom: 1, borderColor: 'divider', px: 2 }}
          >
            <Tab icon={<HotelIcon />} label="Hotels" iconPosition="start" />
            <Tab icon={<PersonIcon />} label="Users" iconPosition="start" />
          </Tabs>

          {/* Hotels Tab */}
          {tabValue === 0 && (
            <Box sx={{ p: 3 }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
                <Typography variant="h6" fontWeight={600}>
                  Hotels List
                </Typography>
                <Button
                  component={Link}
                  to={ROUTES.ADMIN_NEW_HOTEL}
                  variant="contained"
                  startIcon={<AddIcon />}
                >
                  New Hotel
                </Button>
              </Box>

              <TableContainer component={Paper} variant="outlined">
                <Table>
                  <TableHead>
                    <TableRow sx={{ bgcolor: 'background.default' }}>
                      <TableCell sx={{ fontWeight: 600 }}>Name</TableCell>
                      <TableCell sx={{ fontWeight: 600 }}>City</TableCell>
                      <TableCell sx={{ fontWeight: 600 }}>Country</TableCell>
                      <TableCell sx={{ fontWeight: 600 }}>Price/Night</TableCell>
                      <TableCell sx={{ fontWeight: 600 }}>Rating</TableCell>
                      <TableCell sx={{ fontWeight: 600 }} align="right">Actions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {loading ? (
                      Array.from({ length: 5 }).map((_, index) => (
                        <TableRow key={index}>
                          <TableCell><Skeleton /></TableCell>
                          <TableCell><Skeleton /></TableCell>
                          <TableCell><Skeleton /></TableCell>
                          <TableCell><Skeleton /></TableCell>
                          <TableCell><Skeleton /></TableCell>
                          <TableCell><Skeleton /></TableCell>
                        </TableRow>
                      ))
                    ) : hotels.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                          <Typography color="text.secondary">No hotels registered</Typography>
                        </TableCell>
                      </TableRow>
                    ) : (
                      hotels.map((hotel) => (
                        <TableRow key={hotel.id} hover>
                          <TableCell>
                            <Typography variant="body2" fontWeight={500}>
                              {hotel.name}
                            </Typography>
                          </TableCell>
                          <TableCell>{hotel.city}</TableCell>
                          <TableCell>{hotel.country}</TableCell>
                          <TableCell>{formatPrice(hotel.price_per_night || hotel.pricePerNight || 0)}</TableCell>
                          <TableCell>
                            <Chip
                              label={hotel.rating?.toFixed(1) || 'N/A'}
                              size="small"
                              color={hotel.rating >= 4 ? 'success' : hotel.rating >= 3 ? 'warning' : 'default'}
                            />
                          </TableCell>
                          <TableCell align="right">
                            <Tooltip title="View details">
                              <IconButton
                                component={Link}
                                to={`/hotels/${hotel.id}`}
                                size="small"
                              >
                                <HotelIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                            <Tooltip title="Edit">
                              <IconButton
                                component={Link}
                                to={`/admin/hotels/${hotel.id}/edit`}
                                size="small"
                                color="primary"
                              >
                                <EditIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                            <Tooltip title="Delete">
                              <IconButton
                                onClick={() => handleDeleteClick('hotel', hotel)}
                                size="small"
                                color="error"
                              >
                                <DeleteIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>
          )}

          {/* Users Tab */}
          {tabValue === 1 && (
            <Box sx={{ p: 3 }}>
              <Typography variant="h6" fontWeight={600} sx={{ mb: 3 }}>
                Users List
              </Typography>

              <TableContainer component={Paper} variant="outlined">
                <Table>
                  <TableHead>
                    <TableRow sx={{ bgcolor: 'background.default' }}>
                      <TableCell sx={{ fontWeight: 600 }}>ID</TableCell>
                      <TableCell sx={{ fontWeight: 600 }}>Username</TableCell>
                      <TableCell sx={{ fontWeight: 600 }}>Role</TableCell>
                      <TableCell sx={{ fontWeight: 600 }} align="right">Actions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {loading ? (
                      Array.from({ length: 5 }).map((_, index) => (
                        <TableRow key={index}>
                          <TableCell><Skeleton /></TableCell>
                          <TableCell><Skeleton /></TableCell>
                          <TableCell><Skeleton /></TableCell>
                          <TableCell><Skeleton /></TableCell>
                        </TableRow>
                      ))
                    ) : users.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={4} align="center" sx={{ py: 4 }}>
                          <Typography color="text.secondary">No users registered</Typography>
                        </TableCell>
                      </TableRow>
                    ) : (
                      users.map((user) => (
                        <TableRow key={user.id} hover>
                          <TableCell>{user.id}</TableCell>
                          <TableCell>
                            <Typography variant="body2" fontWeight={500}>
                              {user.username}
                            </Typography>
                          </TableCell>
                          <TableCell>
                            <Chip
                              label={user.tipo === USER_ROLES.ADMIN ? 'Admin' : 'Customer'}
                              size="small"
                              color={user.tipo === USER_ROLES.ADMIN ? 'primary' : 'default'}
                              icon={user.tipo === USER_ROLES.ADMIN ? <AdminIcon /> : <PersonIcon />}
                            />
                          </TableCell>
                          <TableCell align="right">
                            <Tooltip title="Delete">
                              <IconButton
                                onClick={() => handleDeleteClick('user', user)}
                                size="small"
                                color="error"
                              >
                                <DeleteIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>
          )}
        </Card>
      </Container>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={deleteDialog.open}
        onClose={() => setDeleteDialog({ open: false, type: '', item: null })}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningIcon color="error" />
          Confirm Deletion
        </DialogTitle>
        <DialogContent>
          <Typography variant="body1">
            Are you sure you want to delete{' '}
            {deleteDialog.type === 'hotel' ? (
              <>the hotel <strong>{deleteDialog.item?.name}</strong></>
            ) : (
              <>user <strong>{deleteDialog.item?.username}</strong></>
            )}
            ?
          </Typography>
          <Alert severity="warning" sx={{ mt: 2 }}>
            This action cannot be undone.
            {deleteDialog.type === 'hotel' && ' All associated reservations will also be deleted.'}
          </Alert>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button
            onClick={() => setDeleteDialog({ open: false, type: '', item: null })}
            variant="outlined"
          >
            Cancel
          </Button>
          <Button
            onClick={handleDeleteConfirm}
            variant="contained"
            color="error"
            disabled={deleteLoading}
          >
            {deleteLoading ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
      >
        <Alert severity={snackbar.severity} onClose={() => setSnackbar({ ...snackbar, open: false })}>
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default Dashboard;
