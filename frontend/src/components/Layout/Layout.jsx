/**
 * Layout Component (plan 13 fases 4/8): skip link + un solo
 * <main id="main-content"> enfocable para el RouteAnnouncer.
 */

import { Box } from '@mui/material';
import Navbar from './Navbar';
import Footer from './Footer';
import SkipLink from '../a11y/SkipLink';

const Layout = ({ children }) => {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh' }}>
      <SkipLink />
      <Navbar />
      <Box
        component="main"
        id="main-content"
        tabIndex={-1}
        sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', outline: 'none' }}
      >
        {children}
      </Box>
      <Footer />
    </Box>
  );
};

export default Layout;
