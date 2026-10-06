/**
 * My Reservations (plan 13 fase 6, F13-06): historial auditable — tabs
 * Upcoming/Past/Cancelled/All con counts, canceladas SIEMPRE visibles,
 * status del backend, mutation con invalidación (nada de setState con la
 * lista filtrada capturando estado viejo).
 */

import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router';
import { Box, Button, Container, Snackbar, Alert, Grid, Skeleton, Tab, Tabs, Typography } from '@mui/material';
import { EventNote as EventNoteIcon } from '@mui/icons-material';
import { useAuth } from '../hooks/useAuth';
import { useMyReservations } from '../hooks/queries';
import { useCancelReservation } from '../hooks/mutations';
import ReservationCard from '../components/reservations/ReservationCard';
import CancelReservationDialog from '../components/reservations/CancelReservationDialog';
import { EmptyState, ErrorState, RouteMeta } from '../components/common';
import { reservationGroup, reservationCounts, RESERVATION_GROUPS } from '../utils/reservations';
import { compareDateOnly } from '../utils/dateOnly';
import { ROUTES } from '../constants';

const TABS = [
  { value: 'upcoming', label: 'Upcoming' },
  { value: 'past', label: 'Past' },
  { value: 'cancelled', label: 'Cancelled' },
  { value: 'all', label: 'All' },
];

const MyReservations = () => {
  const navigate = useNavigate();
  const { user } = useAuth();
  const [tab, setTab] = useState('upcoming');
  const [pendingCancel, setPendingCancel] = useState(null);
  const [cancelledNotice, setCancelledNotice] = useState(false);

  const { data, isPending, isError, error, refetch, fetchNextPage, hasNextPage, isFetchingNextPage, isFetchNextPageError } = useMyReservations(user?.id);
  const cancelMutation = useCancelReservation(user?.id);

  const reservations = useMemo(() => data?.pages.flatMap((page) => page.items) ?? [], [data]);
  const counts = useMemo(() => reservationCounts(reservations), [reservations]);

  const visible = useMemo(() => {
    const filtered =
      tab === 'all' ? reservations : reservations.filter((r) => reservationGroup(r) === tab);
    // Próximas primero dentro de Upcoming; el resto, más recientes primero
    return [...filtered].sort((a, b) =>
      tab === RESERVATION_GROUPS.UPCOMING
        ? compareDateOnly(a.check_in, b.check_in)
        : compareDateOnly(b.check_in, a.check_in),
    );
  }, [reservations, tab]);

  const confirmCancel = async () => {
    try {
      await cancelMutation.mutateAsync(pendingCancel.id);
      setPendingCancel(null);
      setCancelledNotice(true);
    } catch {
      // El error queda en cancelMutation.error y lo muestra el dialog
    }
  };

  return (
    <Box sx={{ bgcolor: 'background.default', minHeight: '100vh' }}>
      <RouteMeta title="My reservations" description="Your booking history, including cancelled stays." />

      <Box sx={{ bgcolor: 'primary.main', py: { xs: 4, md: 6 } }}>
        <Container maxWidth="lg">
          <Typography variant="h3" component="h1" sx={{ color: 'white', fontWeight: 600 }}>
            My reservations
          </Typography>
          <Typography variant="body1" sx={{ color: 'rgba(255,255,255,0.75)' }}>
            Every booking you made, including cancelled ones
          </Typography>
        </Container>
      </Box>

      <Container maxWidth="lg" sx={{ py: 4 }}>
        {isError && !data ? (
          <ErrorState error={error} title="We could not load your reservations" onRetry={refetch} />
        ) : isPending ? (
          <Grid container spacing={2} aria-busy="true">
            {Array.from({ length: 3 }).map((_, index) => (
              <Grid key={index} size={{ xs: 12 }}>
                <Skeleton variant="rectangular" height={140} sx={{ borderRadius: 2 }} />
              </Grid>
            ))}
          </Grid>
        ) : reservations.length === 0 ? (
          <EmptyState
            icon={<EventNoteIcon />}
            title="No reservations yet"
            description="Find a stay and your bookings will appear here."
            actionLabel="Search stays"
            onAction={() => navigate(ROUTES.SEARCH)}
          />
        ) : (
          <>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
              {reservations.length} bookings loaded. Filters and counts apply to loaded bookings.
            </Typography>
            <Tabs
              value={tab}
              onChange={(_event, value) => setTab(value)}
              variant="scrollable"
              allowScrollButtonsMobile
              aria-label="Filter reservations"
              sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}
            >
              {TABS.map(({ value, label }) => (
                <Tab key={value} value={value} label={`${label} (${counts[value]})`} />
              ))}
            </Tabs>

            {visible.length === 0 ? (
              <EmptyState
                title={`Nothing in ${TABS.find((t) => t.value === tab).label.toLowerCase()}`}
                description="Bookings will show up here as their dates or status change."
              />
            ) : (
              <Grid container spacing={2}>
                {visible.map((reservation) => (
                  <Grid key={reservation.id} size={{ xs: 12 }}>
                    <ReservationCard reservation={reservation} onCancel={setPendingCancel} />
                  </Grid>
                ))}
              </Grid>
            )}
            {isFetchNextPageError && <Alert severity="error" sx={{ mt: 2 }}>{error.message}</Alert>}
            {hasNextPage && (
              <Box sx={{ mt: 3, textAlign: 'center' }}>
                <Button variant="outlined" disabled={isFetchingNextPage} onClick={() => fetchNextPage()}>
                  {isFetchingNextPage ? 'Loading…' : isFetchNextPageError ? 'Retry loading more' : 'Load more bookings'}
                </Button>
              </Box>
            )}
          </>
        )}
      </Container>

      <CancelReservationDialog
        reservation={pendingCancel}
        isCancelling={cancelMutation.isPending}
        error={cancelMutation.error}
        onConfirm={confirmCancel}
        onClose={() => {
          setPendingCancel(null);
          cancelMutation.reset();
        }}
      />

      <Snackbar
        open={cancelledNotice}
        autoHideDuration={5000}
        onClose={() => setCancelledNotice(false)}
      >
        <Alert severity="success" onClose={() => setCancelledNotice(false)}>
          Reservation cancelled — it stays in your history.
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default MyReservations;
