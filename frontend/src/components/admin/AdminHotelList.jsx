/**
 * Lista admin de hoteles (plan 13 fase 7): paginación de backend por
 * meta.total; tabla en desktop, cards apiladas en mobile (acciones siempre
 * en viewport); error con retry propio que no arrastra a las otras tabs.
 */

import { Link } from 'react-router';
import {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  IconButton,
  Pagination,
  Paper,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  OpenInNew as ViewIcon,
} from '@mui/icons-material';
import { EmptyState, ErrorState } from '../common';
import { formatMajorAmount } from '../../utils/money';
import { ROUTES } from '../../constants';

const AdminHotelList = ({ query, page, onPageChange, pageSize, onDelete }) => {
  const { data, isPending, isError, error, refetch } = query;

  if (isError) {
    return <ErrorState compact error={error} title="Hotels could not be loaded" onRetry={refetch} />;
  }

  const totalPages = data ? Math.max(1, Math.ceil(data.total / pageSize)) : 1;

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2, gap: 2, flexWrap: 'wrap' }}>
        <Typography variant="h6" component="h2">
          Hotels {data && `(${data.total})`}
        </Typography>
        <Button component={Link} to={ROUTES.ADMIN_NEW_HOTEL} variant="contained" startIcon={<AddIcon />}>
          New hotel
        </Button>
      </Box>

      {isPending ? (
        <Skeleton variant="rectangular" height={320} sx={{ borderRadius: 2 }} aria-busy="true" />
      ) : data.items.length === 0 ? (
        <EmptyState title="No hotels in the catalog" description="Create the first hotel to get started." />
      ) : (
        <>
          {/* Desktop: tabla */}
          <TableContainer component={Paper} variant="outlined" sx={{ display: { xs: 'none', md: 'block' } }}>
            <Table aria-label="Hotels">
              <TableHead>
                <TableRow>
                  <TableCell>Name</TableCell>
                  <TableCell>Location</TableCell>
                  <TableCell align="right">Price / night</TableCell>
                  <TableCell align="right">Rooms</TableCell>
                  <TableCell align="right">Rating</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {data.items.map((hotel) => (
                  <TableRow key={hotel.id} hover>
                    <TableCell>
                      <Typography variant="body2" fontWeight={500}>
                        {hotel.name}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      {hotel.city}, {hotel.country}
                    </TableCell>
                    <TableCell align="right">{formatMajorAmount(hotel.price_per_night)}</TableCell>
                    <TableCell align="right">{hotel.available_rooms}</TableCell>
                    <TableCell align="right">
                      <Chip label={(hotel.rating ?? 0).toFixed(1)} size="small" />
                    </TableCell>
                    <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
                      <Tooltip title="View public page">
                        <IconButton component={Link} to={`/hotels/${hotel.id}`} size="small" aria-label={`View ${hotel.name}`}>
                          <ViewIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Edit">
                        <IconButton
                          component={Link}
                          to={`/admin/hotels/${hotel.id}/edit`}
                          size="small"
                          color="primary"
                          aria-label={`Edit ${hotel.name}`}
                        >
                          <EditIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Delete">
                        <IconButton
                          onClick={() => onDelete(hotel)}
                          size="small"
                          color="error"
                          aria-label={`Delete ${hotel.name}`}
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>

          {/* Mobile: cards con acciones visibles */}
          <Box sx={{ display: { xs: 'grid', md: 'none' }, gap: 1.5 }}>
            {data.items.map((hotel) => (
              <Card key={hotel.id} variant="outlined">
                <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 1, mb: 0.5 }}>
                    <Typography variant="subtitle1" component="h3" fontWeight={600} sx={{ minWidth: 0 }}>
                      {hotel.name}
                    </Typography>
                    <Chip label={(hotel.rating ?? 0).toFixed(1)} size="small" />
                  </Box>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                    {hotel.city}, {hotel.country} · {formatMajorAmount(hotel.price_per_night)} / night ·{' '}
                    {hotel.available_rooms} rooms
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 1 }}>
                    <Button component={Link} to={`/admin/hotels/${hotel.id}/edit`} size="small" variant="outlined">
                      Edit
                    </Button>
                    <Button size="small" color="error" onClick={() => onDelete(hotel)}>
                      Delete
                    </Button>
                  </Box>
                </CardContent>
              </Card>
            ))}
          </Box>

          {totalPages > 1 && (
            <Box sx={{ display: 'flex', justifyContent: 'center', mt: 3 }}>
              <Pagination
                count={totalPages}
                page={page}
                onChange={(_event, value) => onPageChange(value)}
                aria-label="Hotels pages"
              />
            </Box>
          )}
        </>
      )}
    </Box>
  );
};

export default AdminHotelList;
