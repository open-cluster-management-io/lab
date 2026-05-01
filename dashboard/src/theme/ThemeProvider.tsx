import { createTheme, ThemeProvider, CssBaseline } from "@mui/material"
import type { Shadows } from "@mui/material"
import { useMemo, type ReactNode } from "react"

interface ThemeContextProps {
  children: ReactNode
  mode?: "light" | "dark"
}

const headingFont = "'Red Hat Display', 'Helvetica Neue', Arial, sans-serif"

export function MuiThemeProvider({ children, mode = "light" }: ThemeContextProps) {
  const theme = useMemo(
    () =>
      createTheme({
        palette: {
          mode,
          primary: {
            main: "#0066cc",
            dark: "#002952",
            light: "#2b9af3",
          },
          secondary: {
            main: "#ee0000",
            dark: "#c9190b",
          },
          success: { main: "#3e8635" },
          warning: { main: "#f0ab00" },
          error: { main: "#c9190b" },
          info: { main: "#2b9af3" },
          background: {
            default: mode === "light" ? "#f8f9fa" : "#1a1d21",
            paper: mode === "light" ? "#ffffff" : "#343a40",
          },
          text: {
            primary: mode === "light" ? "#1a1d21" : "#f8f9fa",
            secondary: mode === "light" ? "#495057" : "#d1d5da",
          },
          divider: mode === "light" ? "#e5e7ea" : "#495057",
          grey: {
            50: "#f8f9fa",
            100: "#f1f2f3",
            200: "#e5e7ea",
            300: "#d1d5da",
            700: "#495057",
            800: "#343a40",
            900: "#1a1d21",
          },
        },
        typography: {
          fontFamily: "'Red Hat Text', 'Helvetica Neue', Arial, sans-serif",
          h1: { fontFamily: headingFont, fontWeight: 700 },
          h2: { fontFamily: headingFont, fontWeight: 700 },
          h3: { fontFamily: headingFont, fontWeight: 600 },
          h4: { fontFamily: headingFont, fontWeight: 600 },
          h5: { fontFamily: headingFont, fontWeight: 600 },
          h6: { fontFamily: headingFont, fontWeight: 600 },
          subtitle1: { fontWeight: 600 },
          subtitle2: { fontWeight: 600 },
          button: { fontWeight: 600, textTransform: "none" as const },
        },
        shape: {
          borderRadius: 8,
        },
        shadows: [
          "none",
          "0 1px 3px 0 rgba(0,0,0,0.1), 0 1px 2px 0 rgba(0,0,0,0.06)",
          "0 1px 3px 0 rgba(0,0,0,0.1), 0 1px 2px 0 rgba(0,0,0,0.06)",
          "0 4px 6px -1px rgba(0,0,0,0.1), 0 2px 4px -1px rgba(0,0,0,0.06)",
          "0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -2px rgba(0,0,0,0.05)",
          "0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -2px rgba(0,0,0,0.05)",
          "0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -2px rgba(0,0,0,0.05)",
          "0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -2px rgba(0,0,0,0.05)",
          "0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -2px rgba(0,0,0,0.05)",
          ...Array(16).fill("0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -2px rgba(0,0,0,0.05)"),
        ] as Shadows,
        components: {
          MuiButton: {
            styleOverrides: {
              root: {
                borderRadius: 6,
                padding: "10px 20px",
                fontWeight: 600,
                textTransform: "none" as const,
                transition: "all 0.2s ease",
              },
              containedPrimary: {
                background: "linear-gradient(135deg, #0066cc 0%, #2b9af3 100%)",
                "&:hover": {
                  background: "linear-gradient(135deg, #0055aa 0%, #1a8ae3 100%)",
                  transform: "translateY(-1px)",
                  boxShadow: "0 4px 6px -1px rgba(0,0,0,0.1), 0 2px 4px -1px rgba(0,0,0,0.06)",
                },
              },
            },
          },
          MuiPaper: {
            styleOverrides: {
              root: {
                border: "1px solid #e5e7ea",
                boxShadow: "0 1px 3px 0 rgba(0,0,0,0.1), 0 1px 2px 0 rgba(0,0,0,0.06)",
              },
            },
          },
          MuiCard: {
            styleOverrides: {
              root: {
                borderRadius: 12,
                border: "1px solid #e5e7ea",
                transition: "all 0.3s ease",
                "&:hover": {
                  transform: "translateY(-4px)",
                  boxShadow: "0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -2px rgba(0,0,0,0.05)",
                },
              },
            },
          },
          MuiChip: {
            styleOverrides: {
              root: {
                borderRadius: 20,
                fontWeight: 600,
                fontSize: "12px",
                letterSpacing: "0.3px",
              },
              colorSuccess: {
                backgroundColor: "rgba(62,134,53,0.1)",
                color: "#3e8635",
                border: "none",
              },
              colorError: {
                backgroundColor: "rgba(201,25,11,0.1)",
                color: "#c9190b",
                border: "none",
              },
              colorWarning: {
                backgroundColor: "rgba(240,171,0,0.1)",
                color: "#f0ab00",
                border: "none",
              },
              colorInfo: {
                backgroundColor: "rgba(43,154,243,0.1)",
                color: "#2b9af3",
                border: "none",
              },
            },
          },
          MuiTableHead: {
            styleOverrides: {
              root: {
                "& th": {
                  backgroundColor: "#f8f9fa",
                  color: "#343a40",
                  fontWeight: 600,
                  fontSize: "13px",
                  textTransform: "uppercase" as const,
                  letterSpacing: "0.5px",
                  borderBottom: "2px solid #e5e7ea",
                },
              },
            },
          },
          MuiTableCell: {
            styleOverrides: {
              root: {
                padding: "14px 16px",
                fontSize: "14px",
                borderBottom: "1px solid #e5e7ea",
                color: "#343a40",
              },
            },
          },
          MuiTableRow: {
            styleOverrides: {
              root: {
                "&.MuiTableRow-hover:hover": {
                  backgroundColor: "#f8f9fa",
                },
              },
            },
          },
          MuiTextField: {
            styleOverrides: {
              root: {
                "& .MuiOutlinedInput-root": {
                  borderRadius: 6,
                  "&.Mui-focused .MuiOutlinedInput-notchedOutline": {
                    borderColor: "#0066cc",
                    boxShadow: "0 0 0 2px rgba(0,102,204,0.1)",
                  },
                },
              },
            },
          },
          MuiTab: {
            styleOverrides: {
              root: {
                fontWeight: 600,
                textTransform: "none" as const,
                "&.Mui-selected": {
                  color: "#0066cc",
                },
              },
            },
          },
          MuiTabs: {
            styleOverrides: {
              indicator: {
                backgroundColor: "#0066cc",
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
