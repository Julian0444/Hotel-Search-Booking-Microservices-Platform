/**
 * Navigation Bar Component
 * Main navigation with responsive drawer for mobile
 */

import { useState } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import {
  AppBar,
  Box,
  Toolbar,
  IconButton,
  Typography,
  Menu,
  Container,
  Avatar,
  Button,
  Tooltip,
  MenuItem,
  Drawer,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  ListItemIcon,
  Divider,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import {
  Menu as MenuIcon,
  Hotel as HotelIcon,
  EventNote as EventNoteIcon,
  AdminPanelSettings as AdminIcon,
  Logout as LogoutIcon,
  Login as LoginIcon,
  PersonAdd as PersonAddIcon,
  Search as SearchIcon,
} from '@mui/icons-material';
import { useAuth } from '../../context/AuthContext';
import { ROUTES } from '../../constants';
import { getInitials } from '../../utils/helpers';

const Navbar = () => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const navigate = useNavigate();
  const location = useLocation();
  const { user, isAuthenticated, isAdmin, logout } = useAuth();

  const [anchorElUser, setAnchorElUser] = useState(null);
  const [mobileOpen, setMobileOpen] = useState(false);

  const handleOpenUserMenu = (event) => {
    setAnchorElUser(event.currentTarget);
  };

  const handleCloseUserMenu = () => {
    setAnchorElUser(null);
  };

  const handleDrawerToggle = () => {
    setMobileOpen(!mobileOpen);
  };

  const handleLogout = () => {
    logout();
    handleCloseUserMenu();
    navigate(ROUTES.HOME);
  };

  const navItems = [
    { label: 'Home', path: ROUTES.HOME, icon: <HotelIcon /> },
    { label: 'Search', path: ROUTES.SEARCH, icon: <SearchIcon /> },
  ];

  const isActive = (path) => location.pathname === path;

  const drawer = (
    <Box sx={{ width: 280, pt: 2 }}>
      <Box sx={{ px: 3, pb: 2 }}>
        <Typography
          variant="h5"
          sx={{
            fontWeight: 700,
            color: 'primary.main',
            letterSpacing: '-0.02em',
          }}
        >
          StayLux
        </Typography>
        <Typography variant="caption" color="text.secondary">
          Unforgettable Experiences
        </Typography>
      </Box>
      <Divider />
      <List>
        {navItems.map((item) => (
          <ListItem key={item.path} disablePadding>
            <ListItemButton
              component={Link}
              to={item.path}
              onClick={handleDrawerToggle}
              selected={isActive(item.path)}
              sx={{
                mx: 1,
                borderRadius: 2,
                '&.Mui-selected': {
                  bgcolor: 'primary.main',
                  color: 'white',
                  '&:hover': {
                    bgcolor: 'primary.dark',
                  },
                  '& .MuiListItemIcon-root': {
                    color: 'white',
                  },
                },
              }}
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
              onClick={handleDrawerToggle}
              sx={{ mx: 1, borderRadius: 2 }}
            >
              <ListItemIcon sx={{ minWidth: 40 }}>
                <EventNoteIcon />
              </ListItemIcon>
              <ListItemText primary="My Reservations" />
            </ListItemButton>
          </ListItem>
          {isAdmin && (
            <ListItem disablePadding>
              <ListItemButton
                component={Link}
                to={ROUTES.ADMIN}
                onClick={handleDrawerToggle}
                sx={{ mx: 1, borderRadius: 2 }}
              >
                <ListItemIcon sx={{ minWidth: 40 }}>
                  <AdminIcon />
                </ListItemIcon>
                <ListItemText primary="Admin Panel" />
              </ListItemButton>
            </ListItem>
          )}
          <ListItem disablePadding>
            <ListItemButton
              onClick={() => {
                handleLogout();
                handleDrawerToggle();
              }}
              sx={{ mx: 1, borderRadius: 2, color: 'error.main' }}
            >
              <ListItemIcon sx={{ minWidth: 40, color: 'error.main' }}>
                <LogoutIcon />
              </ListItemIcon>
              <ListItemText primary="Sign Out" />
            </ListItemButton>
          </ListItem>
        </List>
      ) : (
        <List>
          <ListItem disablePadding>
            <ListItemButton
              component={Link}
              to={ROUTES.LOGIN}
              onClick={handleDrawerToggle}
              sx={{ mx: 1, borderRadius: 2 }}
            >
              <ListItemIcon sx={{ minWidth: 40 }}>
                <LoginIcon />
              </ListItemIcon>
              <ListItemText primary="Sign In" />
            </ListItemButton>
          </ListItem>
          <ListItem disablePadding>
            <ListItemButton
              component={Link}
              to={ROUTES.REGISTER}
              onClick={handleDrawerToggle}
              sx={{ mx: 1, borderRadius: 2 }}
            >
              <ListItemIcon sx={{ minWidth: 40 }}>
                <PersonAddIcon />
              </ListItemIcon>
              <ListItemText primary="Sign Up" />
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
        sx={{
          bgcolor: 'rgba(255, 255, 255, 0.95)',
          backdropFilter: 'blur(10px)',
          color: 'text.primary',
        }}
      >
        <Container maxWidth="xl">
          <Toolbar disableGutters sx={{ minHeight: { xs: 64, md: 72 } }}>
            {isMobile && (
              <IconButton
                color="inherit"
                aria-label="open drawer"
                edge="start"
                onClick={handleDrawerToggle}
                sx={{ mr: 2 }}
              >
                <MenuIcon />
              </IconButton>
            )}

            <Box
              component={Link}
              to={ROUTES.HOME}
              sx={{
                display: 'flex',
                alignItems: 'center',
                textDecoration: 'none',
                color: 'inherit',
                mr: 4,
              }}
            >
              <HotelIcon sx={{ fontSize: 32, color: 'secondary.main', mr: 1 }} />
              <Box>
                <Typography
                  variant="h5"
                  sx={{
                    fontWeight: 700,
                    color: 'primary.main',
                    lineHeight: 1,
                    letterSpacing: '-0.02em',
                  }}
                >
                  StayLux
                </Typography>
                {!isMobile && (
                  <Typography
                    variant="caption"
                    sx={{
                      color: 'text.secondary',
                      letterSpacing: '0.1em',
                      fontSize: '0.65rem',
                    }}
                  >
                    UNFORGETTABLE EXPERIENCES
                  </Typography>
                )}
              </Box>
            </Box>

            {!isMobile && (
              <Box sx={{ flexGrow: 1, display: 'flex', gap: 1 }}>
                {navItems.map((item) => (
                  <Button
                    key={item.path}
                    component={Link}
                    to={item.path}
                    sx={{
                      color: isActive(item.path) ? 'primary.main' : 'text.secondary',
                      fontWeight: isActive(item.path) ? 600 : 500,
                      position: 'relative',
                      '&::after': {
                        content: '""',
                        position: 'absolute',
                        bottom: 6,
                        left: '50%',
                        transform: 'translateX(-50%)',
                        width: isActive(item.path) ? '60%' : '0%',
                        height: 2,
                        bgcolor: 'secondary.main',
                        transition: 'width 0.3s ease',
                      },
                      '&:hover::after': {
                        width: '60%',
                      },
                    }}
                  >
                    {item.label}
                  </Button>
                ))}
              </Box>
            )}

            <Box sx={{ flexGrow: isMobile ? 1 : 0 }} />

            {!isMobile && (
              <>
                {isAuthenticated ? (
                  <>
                    <Button
                      component={Link}
                      to={ROUTES.RESERVATIONS}
                      startIcon={<EventNoteIcon />}
                      sx={{ mr: 2, color: 'text.secondary' }}
                    >
                      My Reservations
                    </Button>
                    {isAdmin && (
                      <Button
                        component={Link}
                        to={ROUTES.ADMIN}
                        startIcon={<AdminIcon />}
                        sx={{ mr: 2, color: 'secondary.dark' }}
                      >
                        Admin
                      </Button>
                    )}
                    <Tooltip title="Account settings">
                      <IconButton onClick={handleOpenUserMenu} sx={{ p: 0 }}>
                        <Avatar
                          sx={{
                            bgcolor: 'primary.main',
                            width: 40,
                            height: 40,
                            fontSize: '1rem',
                            fontWeight: 600,
                          }}
                        >
                          {getInitials(user?.username)}
                        </Avatar>
                      </IconButton>
                    </Tooltip>
                    <Menu
                      sx={{ mt: '45px' }}
                      anchorEl={anchorElUser}
                      anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
                      keepMounted
                      transformOrigin={{ vertical: 'top', horizontal: 'right' }}
                      open={Boolean(anchorElUser)}
                      onClose={handleCloseUserMenu}
                    >
                      <Box sx={{ px: 2, py: 1 }}>
                        <Typography variant="subtitle2" fontWeight={600}>
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
                        Sign Out
                      </MenuItem>
                    </Menu>
                  </>
                ) : (
                  <Box sx={{ display: 'flex', gap: 1 }}>
                    <Button
                      component={Link}
                      to={ROUTES.LOGIN}
                      variant="outlined"
                      color="primary"
                      sx={{ borderWidth: 2 }}
                    >
                      Sign In
                    </Button>
                    <Button
                      component={Link}
                      to={ROUTES.REGISTER}
                      variant="contained"
                      color="primary"
                    >
                      Sign Up
                    </Button>
                  </Box>
                )}
              </>
            )}

            {isMobile && isAuthenticated && (
              <Avatar
                sx={{
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
        onClose={handleDrawerToggle}
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
