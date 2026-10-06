/**
 * Navigation Bar (plan 13 fase 4): marca propia, links por rol, estado
 * activo con NavLink y drawer mobile accesible que cierra al navegar.
 */

import { useState } from 'react';
import { NavLink, Link, useNavigate, useLocation } from 'react-router';
import {
  AppBar,
  Avatar,
  Box,
  Button,
  Container,
  Divider,
  Drawer,
  IconButton,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Toolbar,
  Typography,
} from '@mui/material';
import {
  Menu as MenuIcon,
  EventNote as EventNoteIcon,
  AdminPanelSettings as AdminIcon,
  Logout as LogoutIcon,
  Login as LoginIcon,
  PersonAdd as PersonAddIcon,
  Search as SearchIcon,
  Home as HomeIcon,
} from '@mui/icons-material';
import { useAuth } from '../../hooks/useAuth';
import { ROUTES } from '../../constants';
import { getInitials } from '../../utils/helpers';
import BrandMark from '../common/BrandMark';

const Navbar = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, isAuthenticated, isAdmin } = useAuth();

  const [anchorElUser, setAnchorElUser] = useState(null);
  const [mobileOpen, setMobileOpen] = useState(false);

  const closeDrawer = () => setMobileOpen(false);

  const handleLogout = () => {
    setAnchorElUser(null);
    closeDrawer();
    navigate(ROUTES.HOME, { replace: true, state: { signOut: true } });
  };

  const navItems = [
    { label: 'Home', path: ROUTES.HOME, icon: <HomeIcon /> },
    { label: 'Search', path: ROUTES.SEARCH, icon: <SearchIcon /> },
  ];

  const isActive = (path) => location.pathname === path;

  const drawer = (
    <Box sx={{ width: 280, pt: 2 }} role="presentation">
      <Box sx={{ px: 3, pb: 2 }}>
        <BrandMark compact />
      </Box>
      <Divider />
      <List>
        {navItems.map((item) => (
          <ListItem key={item.path} disablePadding>
            <ListItemButton
              component={Link}
              to={item.path}
              onClick={closeDrawer}
              selected={isActive(item.path)}
              sx={{ mx: 1, borderRadius: 2, minHeight: 44 }}
            >
              <ListItemIcon sx={{ minWidth: 40 }}>{item.icon}</ListItemIcon>
              <ListItemText primary={item.label} />
            </ListItemButton>
          </ListItem>
        ))}
      </List>
      <Divider sx={{ my: 1 }} />
      {isAuthenticated ? (
        <List>
          <ListItem disablePadding>
            <ListItemButton
              component={Link}
              to={ROUTES.RESERVATIONS}
              onClick={closeDrawer}
              selected={isActive(ROUTES.RESERVATIONS)}
              sx={{ mx: 1, borderRadius: 2, minHeight: 44 }}
            >
              <ListItemIcon sx={{ minWidth: 40 }}>
                <EventNoteIcon />
              </ListItemIcon>
              <ListItemText primary="My reservations" />
            </ListItemButton>
          </ListItem>
          {isAdmin && (
            <ListItem disablePadding>
              <ListItemButton
                component={Link}
                to={ROUTES.ADMIN}
                onClick={closeDrawer}
                selected={isActive(ROUTES.ADMIN)}
                sx={{ mx: 1, borderRadius: 2, minHeight: 44 }}
              >
                <ListItemIcon sx={{ minWidth: 40 }}>
                  <AdminIcon />
                </ListItemIcon>
                <ListItemText primary="Admin panel" />
              </ListItemButton>
            </ListItem>
          )}
          <ListItem disablePadding>
            <ListItemButton onClick={handleLogout} sx={{ mx: 1, borderRadius: 2, minHeight: 44, color: 'error.main' }}>
              <ListItemIcon sx={{ minWidth: 40, color: 'error.main' }}>
                <LogoutIcon />
              </ListItemIcon>
              <ListItemText primary="Sign out" />
            </ListItemButton>
          </ListItem>
        </List>
      ) : (
        <List>
          <ListItem disablePadding>
            <ListItemButton component={Link} to={ROUTES.LOGIN} onClick={closeDrawer} sx={{ mx: 1, borderRadius: 2, minHeight: 44 }}>
              <ListItemIcon sx={{ minWidth: 40 }}>
                <LoginIcon />
              </ListItemIcon>
              <ListItemText primary="Sign in" />
            </ListItemButton>
          </ListItem>
          <ListItem disablePadding>
            <ListItemButton component={Link} to={ROUTES.REGISTER} onClick={closeDrawer} sx={{ mx: 1, borderRadius: 2, minHeight: 44 }}>
              <ListItemIcon sx={{ minWidth: 40 }}>
                <PersonAddIcon />
              </ListItemIcon>
              <ListItemText primary="Create account" />
            </ListItemButton>
          </ListItem>
        </List>
      )}
    </Box>
  );

  return (
    <>
      <AppBar
        position="sticky"
        sx={{ bgcolor: 'rgba(255, 255, 255, 0.96)', backdropFilter: 'blur(8px)', color: 'text.primary' }}
      >
        <Container maxWidth="xl">
          <Toolbar disableGutters sx={{ minHeight: { xs: 64, md: 72 }, gap: 2 }}>
            <IconButton
              color="inherit"
              aria-label="Open navigation menu"
              edge="start"
              onClick={() => setMobileOpen(true)}
              sx={{ display: { md: 'none' } }}
            >
              <MenuIcon />
            </IconButton>

            <BrandMark />

            <Box
              component="nav"
              aria-label="Primary"
              sx={{ flexGrow: 1, display: { xs: 'none', md: 'flex' }, gap: 1, ml: 3 }}
            >
              {navItems.map((item) => (
                <Button
                  key={item.path}
                  component={NavLink}
                  to={item.path}
                  sx={{
                    color: isActive(item.path) ? 'primary.main' : 'text.secondary',
                    fontWeight: isActive(item.path) ? 600 : 500,
                    borderBottom: '2px solid',
                    borderColor: isActive(item.path) ? 'secondary.main' : 'transparent',
                    borderRadius: 0,
                  }}
                >
                  {item.label}
                </Button>
              ))}
            </Box>

            <Box sx={{ flexGrow: { xs: 1, md: 0 } }} />

            <Box sx={{ display: { xs: 'none', md: 'flex' }, alignItems: 'center', gap: 1 }}>
              {isAuthenticated ? (
                <>
                  <Button
                    component={Link}
                    to={ROUTES.RESERVATIONS}
                    startIcon={<EventNoteIcon />}
                    sx={{ color: 'text.secondary' }}
                  >
                    My reservations
                  </Button>
                  {isAdmin && (
                    <Button component={Link} to={ROUTES.ADMIN} startIcon={<AdminIcon />} sx={{ color: 'secondary.dark' }}>
                      Admin
                    </Button>
                  )}
                  <IconButton
                    onClick={(event) => setAnchorElUser(event.currentTarget)}
                    aria-label={`Account menu for ${user?.username}`}
                    sx={{ p: 0.5 }}
                  >
                    <Avatar sx={{ bgcolor: 'primary.main', width: 40, height: 40, fontSize: '1rem', fontWeight: 600 }}>
                      {getInitials(user?.username)}
                    </Avatar>
                  </IconButton>
                  <Menu
                    anchorEl={anchorElUser}
                    anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                    transformOrigin={{ vertical: 'top', horizontal: 'right' }}
                    open={Boolean(anchorElUser)}
                    onClose={() => setAnchorElUser(null)}
                  >
                    <Box sx={{ px: 2, py: 1 }}>
                      <Typography variant="subtitle2" component="p" fontWeight={600}>
                        {user?.username}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {isAdmin ? 'Administrator' : 'Customer'}
                      </Typography>
                    </Box>
                    <Divider />
                    <MenuItem onClick={handleLogout}>
                      <ListItemIcon>
                        <LogoutIcon fontSize="small" />
                      </ListItemIcon>
                      Sign out
                    </MenuItem>
                  </Menu>
                </>
              ) : (
                <>
                  <Button component={Link} to={ROUTES.LOGIN} variant="outlined" color="primary">
                    Sign in
                  </Button>
                  <Button component={Link} to={ROUTES.REGISTER} variant="contained" color="primary">
                    Create account
                  </Button>
                </>
              )}
            </Box>

            {isAuthenticated && (
              <Avatar
                aria-hidden
                sx={{
                  display: { xs: 'flex', md: 'none' },
                  bgcolor: 'primary.main',
                  width: 36,
                  height: 36,
                  fontSize: '0.9rem',
                }}
              >
                {getInitials(user?.username)}
              </Avatar>
            )}
          </Toolbar>
        </Container>
      </AppBar>

      <Drawer
        variant="temporary"
        open={mobileOpen}
        onClose={closeDrawer}
        ModalProps={{ keepMounted: true }}
        sx={{
          display: { xs: 'block', md: 'none' },
          '& .MuiDrawer-paper': { boxSizing: 'border-box', width: 280 },
        }}
      >
        {drawer}
      </Drawer>
    </>
  );
};

export default Navbar;
