/**
 * Admin Dashboard (plan 13 fase 7, RV31/F13-07).
 * Sin Promise.all: hotels/users/services son queries INDEPENDIENTES con
 * retry propio — un error parcial no borra el resto. Tabs que cargan al
 * visitarse; stats derivadas de la data disponible; self-delete bloqueado
 * en la lista; confirmaciones que nombran el recurso.
 */

import { useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  Container,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Snackbar,
  Tab,
  Tabs,
  Typography,
} from '@mui/material';
import {
  Hotel as HotelIcon,
  Person as PersonIcon,
  MonitorHeart as HealthIcon,
} from '@mui/icons-material';
import { useAuth } from '../../hooks/useAuth';
import { useAdminHotels, useAdminUsers, useMicroservicesStatus } from '../../hooks/queries';
import { useDeleteHotel, useDeleteUser } from '../../hooks/mutations';
import AdminHotelList from '../../components/admin/AdminHotelList';
import AdminUserList from '../../components/admin/AdminUserList';
import ServiceHealthGrid from '../../components/admin/ServiceHealthGrid';
import { RouteMeta } from '../../components/common';

const PAGE_SIZE = 10;

const Dashboard = () => {
  const { user } = useAuth();
  const [tab, setTab] = useState('hotels');
  const [hotelsPage, setHotelsPage] = useState(1);
  const [usersPage, setUsersPage] = useState(1);
  const [pendingDelete, setPendingDelete] = useState(null); // {type, item}
  const [notice, setNotice] = useState(null);

  // Cada tab carga al visitarse; el estado de una no pisa a las otras
  const hotelsQuery = useAdminHotels(
    { limit: PAGE_SIZE, offset: (hotelsPage - 1) * PAGE_SIZE },
    { enabled: tab === 'hotels' },
  );
  const usersQuery = useAdminUsers(
    { limit: PAGE_SIZE, offset: (usersPage - 1) * PAGE_SIZE },
    { enabled: tab === 'users' },
  );
  const servicesQuery = useMicroservicesStatus({ enabled: tab === 'services' });

  const deleteHotel = useDeleteHotel();
  const deleteUser = useDeleteUser();
  const deleteMutation = pendingDelete?.type === 'hotel' ? deleteHotel : deleteUser;

  const confirmDelete = async () => {
    const { type, item } = pendingDelete;
    try {
      await (type === 'hotel' ? deleteHotel.mutateAsync(item.id) : deleteUser.mutateAsync(item.id));
      setNotice(`${type === 'hotel' ? 'Hotel' : 'User'} “${item.name || item.username}” deleted`);
      setPendingDelete(null);
    } catch {
      // El error queda en la mutation y se muestra dentro del dialog
    }
  };

  const closeDeleteDialog = () => {
    setPendingDelete(null);
    deleteHotel.reset();
    deleteUser.reset();
  };

  return (
    <Box sx={{ bgcolor: 'background.default', minHeight: '100vh' }}>
      <RouteMeta title="Admin dashboard" description="Manage hotels, users and platform status." />

      <Box sx={{ bgcolor: 'primary.main', py: { xs: 4, md: 5 } }}>
        <Container maxWidth="lg">
          <Typography variant="h3" component="h1" sx={{ color: 'white', fontWeight: 600 }}>
            Admin dashboard
          </Typography>
          <Typography variant="body1" sx={{ color: 'rgba(255,255,255,0.75)' }}>
            Catalog, accounts and real platform health
          </Typography>
        </Container>
      </Box>

      <Container maxWidth="lg" sx={{ py: 4 }}>
        <Card>
          <Tabs
            value={tab}
            onChange={(_event, value) => setTab(value)}
            variant="scrollable"
            allowScrollButtonsMobile
            aria-label="Admin sections"
            sx={{ borderBottom: 1, borderColor: 'divider', px: 2 }}
          >
            <Tab value="hotels" icon={<HotelIcon />} label="Hotels" iconPosition="start" />
            <Tab value="users" icon={<PersonIcon />} label="Users" iconPosition="start" />
            <Tab value="services" icon={<HealthIcon />} label="Services" iconPosition="start" />
          </Tabs>

          <Box sx={{ p: { xs: 2, md: 3 } }}>
            {tab === 'hotels' && (
              <AdminHotelList
                query={hotelsQuery}
                page={hotelsPage}
                onPageChange={setHotelsPage}
                pageSize={PAGE_SIZE}
                onDelete={(item) => setPendingDelete({ type: 'hotel', item })}
              />
            )}
            {tab === 'users' && (
              <AdminUserList
                query={usersQuery}
                page={usersPage}
                onPageChange={setUsersPage}
                pageSize={PAGE_SIZE}
                currentUserId={user?.id}
                onDelete={(item) => setPendingDelete({ type: 'user', item })}
              />
            )}
            {tab === 'services' && <ServiceHealthGrid query={servicesQuery} />}
          </Box>
        </Card>
      </Container>

      {/* Confirmación que nombra el recurso */}
      <Dialog open={!!pendingDelete} onClose={closeDeleteDialog} maxWidth="sm" fullWidth aria-labelledby="delete-title">
        <DialogTitle id="delete-title">
          Delete {pendingDelete?.type === 'hotel' ? 'hotel' : 'user'}?
        </DialogTitle>
        <DialogContent>
          <Typography variant="body1">
            {pendingDelete?.type === 'hotel' ? (
              <>
                The hotel <strong>{pendingDelete?.item.name}</strong> will be removed from the catalog
                and the search index.
              </>
            ) : (
              <>
                The account <strong>{pendingDelete?.item.username}</strong> will be permanently removed.
              </>
            )}
          </Typography>
          <Alert severity="warning" sx={{ mt: 2 }}>
            This action cannot be undone.
          </Alert>
          {deleteMutation?.error && (
            <Alert severity="error" sx={{ mt: 2 }}>
              {deleteMutation.error.message}
            </Alert>
          )}
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={closeDeleteDialog} variant="outlined" disabled={deleteMutation?.isPending}>
            Keep it
          </Button>
          <Button onClick={confirmDelete} variant="contained" color="error" disabled={deleteMutation?.isPending}>
            {deleteMutation?.isPending ? 'Deleting…' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>

      <Snackbar open={!!notice} autoHideDuration={5000} onClose={() => setNotice(null)}>
        <Alert severity="success" onClose={() => setNotice(null)}>
          {notice}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default Dashboard;
