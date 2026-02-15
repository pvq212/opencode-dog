import { defaultDarkTheme, defaultLightTheme } from 'react-admin';
import type { RaThemeOptions } from 'react-admin';

const baseTypography = {
  fontFamily: '"DM Sans", "Helvetica Neue", Helvetica, sans-serif',
  h1: { fontFamily: '"DM Sans", sans-serif', fontWeight: 700, letterSpacing: '-0.02em' },
  h2: { fontFamily: '"DM Sans", sans-serif', fontWeight: 700, letterSpacing: '-0.02em' },
  h3: { fontFamily: '"DM Sans", sans-serif', fontWeight: 600, letterSpacing: '-0.01em' },
  h4: { fontFamily: '"DM Sans", sans-serif', fontWeight: 600 },
  h5: { fontFamily: '"DM Sans", sans-serif', fontWeight: 600 },
  h6: { fontFamily: '"DM Sans", sans-serif', fontWeight: 600 },
  body1: { fontFamily: '"DM Sans", sans-serif', fontSize: '0.9rem' },
  body2: { fontFamily: '"DM Sans", sans-serif', fontSize: '0.85rem' },
  button: { fontFamily: '"DM Sans", sans-serif', fontWeight: 600, textTransform: 'none' as const },
  caption: { fontFamily: '"JetBrains Mono", monospace', fontSize: '0.75rem' },
  overline: { fontFamily: '"JetBrains Mono", monospace', fontSize: '0.7rem', letterSpacing: '0.08em' },
};

export const darkTheme: RaThemeOptions = {
  ...defaultDarkTheme,
  palette: {
    mode: 'dark',
    primary: { main: '#6ee7b7', light: '#a7f3d0', dark: '#34d399', contrastText: '#064e3b' },
    secondary: { main: '#818cf8', light: '#a5b4fc', dark: '#6366f1' },
    error: { main: '#fb7185' },
    warning: { main: '#fbbf24' },
    success: { main: '#34d399' },
    info: { main: '#38bdf8' },
    background: { default: '#0c0e14', paper: '#141720' },
    text: { primary: '#e2e8f0', secondary: '#94a3b8' },
  },
  typography: baseTypography,
  shape: { borderRadius: 10 },
  components: {
    ...defaultDarkTheme.components,
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
          borderColor: 'rgba(148, 163, 184, 0.08)',
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          background: 'linear-gradient(135deg, #141720 0%, #1a1f2e 100%)',
          borderBottom: '1px solid rgba(110, 231, 183, 0.12)',
          boxShadow: '0 1px 24px rgba(0,0,0,0.3)',
        },
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: {
          background: 'linear-gradient(180deg, #0f1219 0%, #141720 100%)',
          borderRight: '1px solid rgba(110, 231, 183, 0.08)',
        },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        root: { borderColor: 'rgba(148, 163, 184, 0.06)' },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: { fontFamily: '"JetBrains Mono", monospace', fontSize: '0.75rem', fontWeight: 500 },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: { borderRadius: 8, padding: '6px 16px' },
        contained: {
          boxShadow: '0 2px 8px rgba(110, 231, 183, 0.2)',
          '&:hover': { boxShadow: '0 4px 16px rgba(110, 231, 183, 0.3)' },
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          border: '1px solid rgba(148, 163, 184, 0.06)',
          boxShadow: '0 4px 24px rgba(0,0,0,0.2)',
        },
      },
    },
    MuiTextField: {
      defaultProps: { variant: 'outlined' as const, size: 'small' as const },
    },
    MuiMenuItem: {
      styleOverrides: {
        root: {
          '&.RaMenuItemLink-active': {
            borderLeft: '3px solid #6ee7b7',
            background: 'rgba(110, 231, 183, 0.08)',
          },
        },
      },
    },
  },
};

export const lightTheme: RaThemeOptions = {
  ...defaultLightTheme,
  palette: {
    mode: 'light',
    primary: { main: '#059669', light: '#34d399', dark: '#047857', contrastText: '#ffffff' },
    secondary: { main: '#4f46e5', light: '#818cf8', dark: '#3730a3' },
    error: { main: '#e11d48' },
    warning: { main: '#d97706' },
    success: { main: '#059669' },
    info: { main: '#0284c7' },
    background: { default: '#f8fafc', paper: '#ffffff' },
    text: { primary: '#1e293b', secondary: '#64748b' },
  },
  typography: baseTypography,
  shape: { borderRadius: 10 },
  components: {
    ...defaultLightTheme.components,
    MuiAppBar: {
      styleOverrides: {
        root: {
          background: 'linear-gradient(135deg, #ffffff 0%, #f8fafc 100%)',
          borderBottom: '1px solid rgba(5, 150, 105, 0.15)',
          boxShadow: '0 1px 12px rgba(0,0,0,0.06)',
          color: '#1e293b',
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: { borderRadius: 8, padding: '6px 16px' },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: { fontFamily: '"JetBrains Mono", monospace', fontSize: '0.75rem', fontWeight: 500 },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          border: '1px solid rgba(0,0,0,0.06)',
          boxShadow: '0 1px 8px rgba(0,0,0,0.04)',
        },
      },
    },
    MuiTextField: {
      defaultProps: { variant: 'outlined' as const, size: 'small' as const },
    },
    MuiMenuItem: {
      styleOverrides: {
        root: {
          '&.RaMenuItemLink-active': {
            borderLeft: '3px solid #059669',
            background: 'rgba(5, 150, 105, 0.06)',
          },
        },
      },
    },
  },
};
