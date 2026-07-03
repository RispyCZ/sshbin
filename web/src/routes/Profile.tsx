import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  FormControl,
  FormControlLabel,
  FormLabel,
  Radio,
  RadioGroup,
  Stack,
  Typography,
} from "@mui/material";
import DeleteForeverIcon from "@mui/icons-material/DeleteForever";
import SaveIcon from "@mui/icons-material/Save";
import { ApiError, api, errMessage } from "../api/client.ts";
import { useNotify } from "../components/NotifyProvider.tsx";
import { useConfirm } from "../components/useConfirm.tsx";

export function Profile({ onSignedOut }: { onSignedOut: () => void }) {
  const navigate = useNavigate();
  const notify = useNotify();
  const { confirm, dialog: confirmDialog } = useConfirm();
  const [loaded, setLoaded] = useState(false);
  const [email, setEmail] = useState("");
  const [visibility, setVisibility] = useState<"public" | "private">("private");
  const [error, setError] = useState("");

  useEffect(() => {
    void (async () => {
      try {
        const p = await api.profile();
        setEmail(p.email);
        setVisibility(p.defaultPublic ? "public" : "private");
        setLoaded(true);
      } catch (err) {
        setError(err instanceof ApiError ? err.message : "Could not load profile.");
      }
    })();
  }, []);

  async function save() {
    try {
      await api.saveProfile(visibility === "public");
      notify("Preferences saved");
    } catch (err) {
      notify(errMessage(err, "Could not save preferences."), "error");
    }
  }

  async function deleteAll() {
    const ok = await confirm({
      title: "Delete all data",
      message: "Delete all your shares and sign out everywhere? This cannot be undone.",
      confirmLabel: "Delete all",
      danger: true,
    });
    if (!ok) return;
    try {
      await api.deleteAllData();
      notify("All data deleted");
      onSignedOut();
      void navigate("/login", { replace: true });
    } catch (err) {
      notify(errMessage(err, "Could not delete data."), "error");
    }
  }

  if (!loaded) {
    return error ? (
      <Alert severity="error">{error}</Alert>
    ) : (
      <Box sx={{ display: "flex", justifyContent: "center", py: 6 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Stack spacing={3} sx={{ maxWidth: 480 }}>
      <Typography variant="h5" component="h1">
        Profile
      </Typography>
      <Typography variant="body2" color="text.secondary">
        {email}
      </Typography>

      <Card variant="outlined">
        <CardContent>
          <Stack spacing={2}>
            <FormControl>
              <FormLabel>Default visibility for new shares</FormLabel>
              <RadioGroup
                value={visibility}
                onChange={(e) => setVisibility(e.target.value as "public" | "private")}
              >
                <FormControlLabel
                  value="private"
                  control={<Radio />}
                  label="Private — only people I list"
                />
                <FormControlLabel
                  value="public"
                  control={<Radio />}
                  label="Public — anyone with the link"
                />
              </RadioGroup>
            </FormControl>
            <Box>
              <Button variant="contained" startIcon={<SaveIcon />} onClick={() => void save()}>
                Save
              </Button>
            </Box>
          </Stack>
        </CardContent>
      </Card>

      <Card variant="outlined" sx={{ borderColor: "error.main" }}>
        <CardContent>
          <Stack spacing={1.5}>
            <Typography variant="h6" color="error">
              Danger zone
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Delete all your shares and sign out from all devices. This cannot be undone.
            </Typography>
            <Box>
              <Button
                variant="outlined"
                color="error"
                startIcon={<DeleteForeverIcon />}
                onClick={() => void deleteAll()}
              >
                Delete all my data
              </Button>
            </Box>
          </Stack>
        </CardContent>
      </Card>

      {confirmDialog}
    </Stack>
  );
}
