/**
 * Lista admin de usuarios (plan 13 fase 7, RV31): self-delete DESHABILITADO
 * comparando IDs canónicos string, con el motivo visible; paginación de
 * backend por meta.total; tabla desktop / cards mobile.
 */

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
  Delete as DeleteIcon,
  AdminPanelSettings as AdminIcon,
  Person as PersonIcon,
} from '@mui/icons-material';
import { EmptyState, ErrorState } from '../common';
import { USER_ROLES } from '../../constants';

const SELF_DELETE_REASON = 'You cannot delete your own account while signed in';

const AdminUserList = ({ query, page, onPageChange, pageSize, currentUserId, onDelete }) => {
  const { data, isPending, isError, error, refetch } = query;

  if (isError) {
    return <ErrorState compact error={error} title="Users could not be loaded" onRetry={refetch} />;
  }

  const totalPages = data ? Math.max(1, Math.ceil(data.total / pageSize)) : 1;
  const isSelf = (user) => String(user.id) === String(currentUserId);

  return (
    <Box>
      <Typography variant="h6" component="h2" sx={{ mb: 2 }}>
        Users {data && `(${data.total})`}
      </Typography>

      {isPending ? (
        <Skeleton variant="rectangular" height={280} sx={{ borderRadius: 2 }} aria-busy="true" />
      ) : data.items.length === 0 ? (
        <EmptyState title="No users" description="Registered accounts will appear here." />
      ) : (
        <>
          <TableContainer component={Paper} variant="outlined" sx={{ display: { xs: 'none', md: 'block' } }}>
            <Table aria-label="Users">
              <TableHead>
                <TableRow>
                  <TableCell>ID</TableCell>
                  <TableCell>Username</TableCell>
                  <TableCell>Role</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {data.items.map((user) => (
                  <TableRow key={user.id} hover>
                    <TableCell sx={{ fontFamily: 'monospace' }}>{user.id}</TableCell>
                    <TableCell>
                      <Typography variant="body2" fontWeight={500}>
                        {user.username}
                        {isSelf(user) && (
                          <Typography component="span" variant="caption" color="text.secondary" sx={{ ml: 1 }}>
                            (you)
                          </Typography>
                        )}
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
                      <Tooltip title={isSelf(user) ? SELF_DELETE_REASON : 'Delete user'}>
                        {/* span: el tooltip necesita un target habilitado */}
                        <span>
                          <IconButton
                            onClick={() => onDelete(user)}
                            size="small"
                            color="error"
                            disabled={isSelf(user)}
                            aria-label={isSelf(user) ? SELF_DELETE_REASON : `Delete ${user.username}`}
                          >
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </span>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>

          <Box sx={{ display: { xs: 'grid', md: 'none' }, gap: 1.5 }}>
            {data.items.map((user) => (
              <Card key={user.id} variant="outlined">
                <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 1 }}>
                    <Box sx={{ minWidth: 0 }}>
                      <Typography variant="subtitle2" component="h3" fontWeight={600}>
                        {user.username} {isSelf(user) && '(you)'}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        #{user.id} · {user.tipo === USER_ROLES.ADMIN ? 'Admin' : 'Customer'}
                      </Typography>
                    </Box>
                    <Button
                      size="small"
                      color="error"
                      disabled={isSelf(user)}
                      onClick={() => onDelete(user)}
                    >
                      Delete
                    </Button>
                  </Box>
                  {isSelf(user) && (
                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                      {SELF_DELETE_REASON}
                    </Typography>
                  )}
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
                aria-label="Users pages"
              />
            </Box>
          )}
        </>
      )}
    </Box>
  );
};

export default AdminUserList;
