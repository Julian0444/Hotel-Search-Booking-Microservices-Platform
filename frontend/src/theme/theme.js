/**
 * Theme StayLux (plan 13 fase 4) — "hospitality editorial argentina".
 * Tokens formalizados: ink navy + antique gold sobre warm parchment, con
 * terracotta/sage SOLO como acentos semánticos. Bordes finos, radios
 * moderados, sombras cortas; serif en headings con clamp() para mobile.
 * Sin gradientes decorativos ni animaciones constantes; el smooth scroll
 * respeta prefers-reduced-motion.
 */

import { createTheme, alpha } from '@mui/material/styles';

// --- Design tokens ---
export const tokens = {
  ink: '#1a365d', // navy: marca y texto de énfasis
  inkDark: '#0f2847',
  inkLight: '#2d4a7c',
  gold: '#c6a961', // antique gold: firma y acentos
  goldLight: '#d4bc7d',
  // Bronce profundo: variante del gold para TEXTO sobre fondos claros —
  // 5.6:1 sobre blanco y 5.1:1 sobre parchment (AA); el #a88b45 original
  // quedaba en ~3:1 y fallaba color-contrast en Lighthouse/axe
  goldDark: '#7f6429',
  surface: '#f7f5f2', // warm parchment
  surfaceElevated: '#ffffff',
  text: '#22242a',
  muted: '#5a5f6a',
  terracotta: '#b4552d', // acento semántico: error/destructivo
  sage: '#5a7a5e', // acento semántico: éxito
  contentWidth: 1200,
  radius: { sm: 6, md: 10, lg: 14 },
  shadow: {
    card: '0 1px 3px rgba(26, 54, 93, 0.10), 0 4px 14px rgba(26, 54, 93, 0.06)',
    cardHover: '0 2px 6px rgba(26, 54, 93, 0.12), 0 8px 22px rgba(26, 54, 93, 0.10)',
  },
};

const serif = '"Cormorant Garamond", Georgia, "Times New Roman", serif';
const sans = '"Source Sans 3", -apple-system, "Segoe UI", "Helvetica Neue", Arial, sans-serif';

const theme = createTheme({
  palette: {
    primary: { main: tokens.ink, light: tokens.inkLight, dark: tokens.inkDark, contrastText: '#ffffff' },
    secondary: { main: tokens.gold, light: tokens.goldLight, dark: tokens.goldDark, contrastText: tokens.ink },
    background: { default: tokens.surface, paper: tokens.surfaceElevated },
    text: { primary: tokens.text, secondary: tokens.muted },
    error: { main: tokens.terracotta },
    success: { main: tokens.sage },
    warning: { main: '#a2681f' },
    info: { main: '#2b6cb0' },
  },
  typography: {
    fontFamily: sans,
    h1: { fontFamily: serif, fontWeight: 600, letterSpacing: '-0.02em', fontSize: 'clamp(2.2rem, 5vw, 3.6rem)' },
    h2: { fontFamily: serif, fontWeight: 600, letterSpacing: '-0.01em', fontSize: 'clamp(1.9rem, 4vw, 2.8rem)' },
    h3: { fontFamily: serif, fontWeight: 600, fontSize: 'clamp(1.6rem, 3vw, 2.2rem)' },
    h4: { fontFamily: serif, fontWeight: 600, fontSize: 'clamp(1.35rem, 2.5vw, 1.8rem)' },
    h5: { fontFamily: sans, fontWeight: 600 },
    h6: { fontFamily: sans, fontWeight: 600 },
    subtitle1: { fontWeight: 500, letterSpacing: '0.01em' },
    subtitle2: { fontWeight: 600, letterSpacing: '0.01em' },
    body1: { lineHeight: 1.7 },
    body2: { lineHeight: 1.6 },
    button: { fontWeight: 600, letterSpacing: '0.02em', textTransform: 'none' },
    overline: { fontWeight: 600, letterSpacing: '0.16em', textTransform: 'uppercase' },
  },
  shape: {
    borderRadius: tokens.radius.sm,
  },
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        html: {
          '@media (prefers-reduced-motion: no-preference)': {
            scrollBehavior: 'smooth',
          },
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius: tokens.radius.sm,
          padding: '10px 22px',
          minHeight: 44,
          fontSize: '0.9375rem',
        },
        contained: {
          boxShadow: 'none',
          '&:hover': { boxShadow: tokens.shadow.card },
        },
        outlined: {
          borderWidth: 1.5,
          '&:hover': { borderWidth: 1.5, backgroundColor: alpha(tokens.ink, 0.04) },
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: tokens.radius.md,
          border: `1px solid ${alpha(tokens.ink, 0.08)}`,
          boxShadow: tokens.shadow.card,
          '&:hover': { boxShadow: tokens.shadow.cardHover },
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: { backgroundImage: 'none' },
        elevation1: { boxShadow: tokens.shadow.card },
        elevation2: { boxShadow: tokens.shadow.card },
        elevation3: { boxShadow: tokens.shadow.cardHover },
        elevation6: { boxShadow: tokens.shadow.cardHover },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: { boxShadow: `0 1px 0 ${alpha(tokens.ink, 0.1)}` },
      },
    },
    MuiTextField: {
      styleOverrides: {
        root: {
          '& .MuiOutlinedInput-root': { borderRadius: tokens.radius.sm },
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: { fontWeight: 500 },
        colorPrimary: { backgroundColor: alpha(tokens.ink, 0.1), color: tokens.ink },
        colorSecondary: { backgroundColor: alpha(tokens.gold, 0.16), color: tokens.goldDark },
      },
    },
    MuiRating: {
      styleOverrides: {
        iconFilled: { color: tokens.gold },
        iconHover: { color: tokens.goldLight },
      },
    },
    MuiDialog: {
      styleOverrides: {
        paper: { borderRadius: tokens.radius.lg },
      },
    },
    MuiDialogTitle: {
      styleOverrides: {
        root: { fontFamily: serif, fontWeight: 600, fontSize: '1.5rem' },
      },
    },
    MuiTab: {
      styleOverrides: {
        root: { textTransform: 'none', fontWeight: 500, fontSize: '0.9375rem', minHeight: 48 },
      },
    },
    MuiAlert: {
      styleOverrides: {
        root: { borderRadius: tokens.radius.sm },
      },
    },
    MuiTableHead: {
      styleOverrides: {
        root: {
          '& .MuiTableCell-root': { fontWeight: 600, backgroundColor: tokens.surface },
        },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        root: { borderColor: alpha(tokens.ink, 0.08) },
      },
    },
    MuiDivider: {
      styleOverrides: {
        root: { borderColor: alpha(tokens.ink, 0.08) },
      },
    },
  },
});

export default theme;
