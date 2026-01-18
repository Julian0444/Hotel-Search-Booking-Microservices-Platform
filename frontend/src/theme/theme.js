/**
 * Material UI Theme Configuration
 * Elegant luxury hotel aesthetic with distinctive typography
 */

import { createTheme, alpha } from '@mui/material/styles';

// Color palette inspired by luxury hotels
const colors = {
  primary: {
    main: '#1a365d',      // Deep navy blue
    light: '#2d4a7c',
    dark: '#0f2847',
    contrastText: '#ffffff',
  },
  secondary: {
    main: '#c6a961',      // Elegant gold
    light: '#d4bc7d',
    dark: '#a88b45',
    contrastText: '#1a365d',
  },
  background: {
    default: '#f7f5f2',   // Warm off-white
    paper: '#ffffff',
  },
  text: {
    primary: '#1a1a1a',
    secondary: '#5a5a5a',
  },
  error: {
    main: '#c53030',
  },
  success: {
    main: '#276749',
  },
  warning: {
    main: '#c05621',
  },
  info: {
    main: '#2b6cb0',
  },
};

const theme = createTheme({
  palette: colors,
  typography: {
    fontFamily: '"Source Sans 3", "Helvetica Neue", Arial, sans-serif',
    h1: {
      fontFamily: '"Cormorant Garamond", Georgia, serif',
      fontWeight: 600,
      letterSpacing: '-0.02em',
    },
    h2: {
      fontFamily: '"Cormorant Garamond", Georgia, serif',
      fontWeight: 600,
      letterSpacing: '-0.01em',
    },
    h3: {
      fontFamily: '"Cormorant Garamond", Georgia, serif',
      fontWeight: 600,
    },
    h4: {
      fontFamily: '"Cormorant Garamond", Georgia, serif',
      fontWeight: 500,
    },
    h5: {
      fontFamily: '"Source Sans 3", sans-serif',
      fontWeight: 600,
    },
    h6: {
      fontFamily: '"Source Sans 3", sans-serif',
      fontWeight: 600,
    },
    subtitle1: {
      fontWeight: 500,
      letterSpacing: '0.01em',
    },
    subtitle2: {
      fontWeight: 600,
      letterSpacing: '0.01em',
    },
    body1: {
      lineHeight: 1.7,
    },
    body2: {
      lineHeight: 1.6,
    },
    button: {
      fontWeight: 600,
      letterSpacing: '0.03em',
      textTransform: 'none',
    },
    overline: {
      fontWeight: 600,
      letterSpacing: '0.15em',
      textTransform: 'uppercase',
    },
  },
  shape: {
    borderRadius: 8,
  },
  shadows: [
    'none',
    '0 1px 3px rgba(26, 54, 93, 0.08)',
    '0 2px 6px rgba(26, 54, 93, 0.1)',
    '0 4px 12px rgba(26, 54, 93, 0.12)',
    '0 6px 16px rgba(26, 54, 93, 0.14)',
    '0 8px 24px rgba(26, 54, 93, 0.16)',
    '0 12px 32px rgba(26, 54, 93, 0.18)',
    '0 16px 40px rgba(26, 54, 93, 0.2)',
    '0 20px 48px rgba(26, 54, 93, 0.22)',
    '0 24px 56px rgba(26, 54, 93, 0.24)',
    '0 28px 64px rgba(26, 54, 93, 0.26)',
    '0 32px 72px rgba(26, 54, 93, 0.28)',
    '0 36px 80px rgba(26, 54, 93, 0.3)',
    '0 40px 88px rgba(26, 54, 93, 0.32)',
    '0 44px 96px rgba(26, 54, 93, 0.34)',
    '0 48px 104px rgba(26, 54, 93, 0.36)',
    '0 52px 112px rgba(26, 54, 93, 0.38)',
    '0 56px 120px rgba(26, 54, 93, 0.4)',
    '0 60px 128px rgba(26, 54, 93, 0.42)',
    '0 64px 136px rgba(26, 54, 93, 0.44)',
    '0 68px 144px rgba(26, 54, 93, 0.46)',
    '0 72px 152px rgba(26, 54, 93, 0.48)',
    '0 76px 160px rgba(26, 54, 93, 0.5)',
    '0 80px 168px rgba(26, 54, 93, 0.52)',
    '0 84px 176px rgba(26, 54, 93, 0.54)',
  ],
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: {
          scrollBehavior: 'smooth',
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius: 6,
          padding: '10px 24px',
          fontSize: '0.9375rem',
          transition: 'all 0.2s ease-in-out',
        },
        contained: {
          boxShadow: '0 2px 8px rgba(26, 54, 93, 0.2)',
          '&:hover': {
            boxShadow: '0 4px 16px rgba(26, 54, 93, 0.3)',
            transform: 'translateY(-1px)',
          },
        },
        containedPrimary: {
          background: `linear-gradient(135deg, ${colors.primary.main} 0%, ${colors.primary.dark} 100%)`,
          '&:hover': {
            background: `linear-gradient(135deg, ${colors.primary.light} 0%, ${colors.primary.main} 100%)`,
          },
        },
        containedSecondary: {
          background: `linear-gradient(135deg, ${colors.secondary.main} 0%, ${colors.secondary.dark} 100%)`,
          '&:hover': {
            background: `linear-gradient(135deg, ${colors.secondary.light} 0%, ${colors.secondary.main} 100%)`,
          },
        },
        outlined: {
          borderWidth: 2,
          '&:hover': {
            borderWidth: 2,
          },
        },
        outlinedPrimary: {
          '&:hover': {
            backgroundColor: alpha(colors.primary.main, 0.04),
          },
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 12,
          boxShadow: '0 4px 20px rgba(26, 54, 93, 0.08)',
          transition: 'all 0.3s ease-in-out',
          '&:hover': {
            boxShadow: '0 8px 40px rgba(26, 54, 93, 0.12)',
          },
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
        },
        elevation1: {
          boxShadow: '0 2px 8px rgba(26, 54, 93, 0.08)',
        },
        elevation2: {
          boxShadow: '0 4px 16px rgba(26, 54, 93, 0.1)',
        },
        elevation3: {
          boxShadow: '0 6px 24px rgba(26, 54, 93, 0.12)',
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          boxShadow: '0 2px 12px rgba(26, 54, 93, 0.08)',
        },
      },
    },
    MuiTextField: {
      styleOverrides: {
        root: {
          '& .MuiOutlinedInput-root': {
            borderRadius: 8,
            transition: 'all 0.2s ease-in-out',
            '&:hover': {
              '& .MuiOutlinedInput-notchedOutline': {
                borderColor: colors.primary.light,
              },
            },
            '&.Mui-focused': {
              '& .MuiOutlinedInput-notchedOutline': {
                borderWidth: 2,
              },
            },
          },
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          fontWeight: 500,
        },
        colorPrimary: {
          backgroundColor: alpha(colors.primary.main, 0.1),
          color: colors.primary.main,
        },
        colorSecondary: {
          backgroundColor: alpha(colors.secondary.main, 0.15),
          color: colors.secondary.dark,
        },
      },
    },
    MuiRating: {
      styleOverrides: {
        iconFilled: {
          color: colors.secondary.main,
        },
        iconHover: {
          color: colors.secondary.light,
        },
      },
    },
    MuiDialog: {
      styleOverrides: {
        paper: {
          borderRadius: 16,
        },
      },
    },
    MuiDialogTitle: {
      styleOverrides: {
        root: {
          fontFamily: '"Cormorant Garamond", Georgia, serif',
          fontWeight: 600,
          fontSize: '1.5rem',
        },
      },
    },
    MuiTab: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          fontWeight: 500,
          fontSize: '0.9375rem',
        },
      },
    },
    MuiAlert: {
      styleOverrides: {
        root: {
          borderRadius: 8,
        },
        standardSuccess: {
          backgroundColor: alpha(colors.success.main, 0.1),
          color: colors.success.main,
        },
        standardError: {
          backgroundColor: alpha(colors.error.main, 0.1),
          color: colors.error.main,
        },
        standardWarning: {
          backgroundColor: alpha(colors.warning.main, 0.1),
          color: colors.warning.main,
        },
        standardInfo: {
          backgroundColor: alpha(colors.info.main, 0.1),
          color: colors.info.main,
        },
      },
    },
    MuiAvatar: {
      styleOverrides: {
        root: {
          fontWeight: 600,
        },
      },
    },
    MuiTableHead: {
      styleOverrides: {
        root: {
          '& .MuiTableCell-root': {
            fontWeight: 600,
            backgroundColor: colors.background.default,
          },
        },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        root: {
          borderColor: alpha(colors.primary.main, 0.08),
        },
      },
    },
    MuiDivider: {
      styleOverrides: {
        root: {
          borderColor: alpha(colors.primary.main, 0.08),
        },
      },
    },
  },
});

export default theme;
