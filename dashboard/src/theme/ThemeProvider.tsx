import { createTheme, ThemeProvider, CssBaseline } from "@mui/material"
import { useMemo, type ReactNode } from "react"

interface ThemeContextProps {
  children: ReactNode
  mode?: "light" | "dark"
}

export function MuiThemeProvider({ children, mode = "light" }: ThemeContextProps) {
  const theme = useMemo(
    () =>
      createTheme({
        palette: {
          mode,
          primary: {
            main: "#6b46c1",
          },
          secondary: {
            main: "#a78bfa",
          },
          background: {
            default: mode === "light" ? "#f8f7fc" : "#1a1429",
            paper: mode === "light" ? "#ffffff" : "#281e3d",
          },
        },
        typography: {
          fontFamily: "Inter, system-ui, sans-serif",
        },
        shape: {
          borderRadius: 8,
        },
        components: {
          MuiButton: {
            styleOverrides: {
              root: {
                borderRadius: 6,
                padding: "8px 16px",
                fontWeight: 500,
              },
            },
          },
          MuiTextField: {
            styleOverrides: {
              root: {
                "& .MuiOutlinedInput-root": {
                  borderRadius: 6,
                },
              },
            },
          },
          MuiCard: {
            styleOverrides: {
              root: {
                borderRadius: 12,
              },
            },
          },
          MuiCssBaseline: {
            styleOverrides: {
              html: {
                height: "100%",
                width: "100%",
              },
              body: {
                height: "100%",
                width: "100%",
                margin: 0,
                padding: 0,
                display: "flex",
                flexDirection: "column",
              },
              "#root": {
                height: "100%",
                width: "100%",
                display: "flex",
                flexDirection: "column",
              },
            },
          },
        },
      }),
    [mode],
  )

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      {children}
    </ThemeProvider>
  )
}
