/**
 * RouteMeta (plan 13 fase 8): título y description por pantalla. React 19
 * hoistea <title>/<meta> al head nativamente — sin dependencias.
 */

const RouteMeta = ({ title, description }) => (
  <>
    <title>{title ? `${title} · StayLux` : 'StayLux'}</title>
    {description && <meta name="description" content={description} />}
  </>
);

export default RouteMeta;
