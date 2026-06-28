import { useState } from "react";
import { Link as RouterLink } from "react-router-dom";
import { Avatar, Button, Divider, ListItemIcon, Menu, MenuItem } from "@mui/material";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import FolderIcon from "@mui/icons-material/Folder";
import LogoutIcon from "@mui/icons-material/Logout";
import PersonIcon from "@mui/icons-material/Person";

export function UserMenu({ email, onLogout }: { email: string; onLogout: () => void }) {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const close = () => setAnchor(null);

  return (
    <>
      <Button
        color="inherit"
        onClick={(e) => setAnchor(e.currentTarget)}
        endIcon={<ExpandMoreIcon />}
        startIcon={
          <Avatar
            sx={{
              width: 24,
              height: 24,
              fontSize: 14,
              bgcolor: "primary.main",
              color: "primary.contrastText",
            }}
          >
            {email[0]?.toUpperCase()}
          </Avatar>
        }
        sx={{ textTransform: "none" }}
      >
        {email}
      </Button>
      <Menu
        anchorEl={anchor}
        open={Boolean(anchor)}
        onClose={close}
        anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
        transformOrigin={{ vertical: "top", horizontal: "right" }}
      >
        <MenuItem component={RouterLink} to="/" onClick={close}>
          <ListItemIcon>
            <FolderIcon fontSize="small" />
          </ListItemIcon>
          My shares
        </MenuItem>
        <MenuItem component={RouterLink} to="/profile" onClick={close}>
          <ListItemIcon>
            <PersonIcon fontSize="small" />
          </ListItemIcon>
          Profile
        </MenuItem>
        <Divider />
        <MenuItem
          onClick={() => {
            close();
            onLogout();
          }}
        >
          <ListItemIcon>
            <LogoutIcon fontSize="small" />
          </ListItemIcon>
          Sign out
        </MenuItem>
      </Menu>
    </>
  );
}
