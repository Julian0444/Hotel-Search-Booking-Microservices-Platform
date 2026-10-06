/**
 * Panel de servicios (plan 13 fase 7): READ-ONLY sobre el shape real del
 * plan 11 ({services, summary}) — readiness, latencia por instancia y última
 * actualización. Observabilidad, no control plane: sin scale/restart/logs.
 */

import { Box, Card, CardContent, Chip, Grid, Skeleton, Typography } from '@mui/material';
import { Circle as CircleIcon } from '@mui/icons-material';
import { ErrorState } from '../common';

const STATUS_COLORS = { up: 'success', degraded: 'warning', down: 'error' };

const ServiceCard = ({ service }) => (
  <Card component="article" aria-label={`${service.name} status`}>
    <CardContent sx={{ p: 2.5 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
        <Typography variant="subtitle1" component="p" fontWeight={600} sx={{ fontFamily: 'monospace' }}>
          {service.name}
        </Typography>
        <Chip
          size="small"
          color={STATUS_COLORS[service.status] ?? 'default'}
          label={service.status}
          icon={<CircleIcon sx={{ fontSize: 10 }} />}
        />
      </Box>
      {service.load_balanced && (
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
          Load balanced · {service.instances.length} instances
        </Typography>
      )}
      <Box component="ul" sx={{ listStyle: 'none', m: 0, p: 0, display: 'grid', gap: 0.75 }}>
        {service.instances.map((instance) => (
          <Box
            key={instance.url}
            component="li"
            sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 1 }}
          >
            <Typography variant="body2" sx={{ fontFamily: 'monospace', minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {instance.name}
            </Typography>
            <Typography
              variant="caption"
              sx={{ color: instance.status === 'up' ? 'success.main' : 'error.main', flexShrink: 0 }}
            >
              {instance.status === 'up' ? `${instance.latency_ms} ms` : instance.error || 'down'}
            </Typography>
          </Box>
        ))}
      </Box>
    </CardContent>
  </Card>
);

const ServiceHealthGrid = ({ query }) => {
  const { data, isPending, isError, error, refetch, dataUpdatedAt } = query;

  if (isError) {
    return <ErrorState compact error={error} title="Platform status is unavailable" onRetry={refetch} />;
  }

  if (isPending) {
    return (
      <Grid container spacing={2} aria-busy="true">
        {Array.from({ length: 3 }).map((_, index) => (
          <Grid key={index} size={{ xs: 12, sm: 6, md: 4 }}>
            <Skeleton variant="rectangular" height={150} sx={{ borderRadius: 2 }} />
          </Grid>
        ))}
      </Grid>
    );
  }

  const { services, summary } = data;

  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 1, mb: 2 }}>
        <Typography variant="body2" color="text.secondary">
          {summary.healthy_services}/{summary.total_services} services healthy ·{' '}
          {summary.total_instances} instances probed via <code>/readyz</code>
        </Typography>
        <Typography variant="caption" color="text.secondary">
          Read-only observability · updated {new Date(dataUpdatedAt).toLocaleTimeString()}
        </Typography>
      </Box>
      <Grid container spacing={2}>
        {services.map((service) => (
          <Grid key={service.name} size={{ xs: 12, sm: 6, md: 4 }}>
            <ServiceCard service={service} />
          </Grid>
        ))}
      </Grid>
    </Box>
  );
};

export default ServiceHealthGrid;
