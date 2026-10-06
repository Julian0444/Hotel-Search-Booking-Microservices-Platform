/**
 * Fallback de Suspense por ruta (plan 13 fase 9, FE2): mientras baja un
 * chunk lazy se mantiene el layout completo — navbar y footer estables, un
 * loader accesible en el main — en vez de un spinner full-screen que haga
 * saltar la página.
 */

import Layout from '../Layout/Layout';
import AppLoader from './AppLoader';

const RouteFallback = () => (
  <Layout>
    <AppLoader label="Loading page" />
  </Layout>
);

export default RouteFallback;
