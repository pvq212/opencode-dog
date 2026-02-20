import { Layout, Menu, usePermissions } from 'react-admin';
import DashboardIcon from '@mui/icons-material/SpaceDashboard';
import FolderIcon from '@mui/icons-material/FolderOpen';
import KeyIcon from '@mui/icons-material/VpnKey';
import TaskIcon from '@mui/icons-material/Assignment';
import SettingsIcon from '@mui/icons-material/Tune';
import ServerIcon from '@mui/icons-material/Dns';
import PeopleIcon from '@mui/icons-material/PeopleAlt';
import MenuBookIcon from '@mui/icons-material/MenuBook';

const AppMenu = () => {
  const { permissions } = usePermissions();
  const role = permissions as string;

  return (
    <Menu>
      <Menu.DashboardItem primaryText="Dashboard" leftIcon={<DashboardIcon />} />
      <Menu.ResourceItem name="projects" primaryText="Projects" leftIcon={<FolderIcon />} />
      {(role === 'admin') && (
        <Menu.ResourceItem name="ssh-keys" primaryText="SSH Keys" leftIcon={<KeyIcon />} />
      )}
      <Menu.ResourceItem name="tasks" primaryText="Tasks" leftIcon={<TaskIcon />} />
      {(role === 'admin') && (
        <>
          <Menu.ResourceItem name="settings" primaryText="Settings" leftIcon={<SettingsIcon />} />
          <Menu.ResourceItem name="mcp-servers" primaryText="MCP Servers" leftIcon={<ServerIcon />} />
          <Menu.ResourceItem name="users" primaryText="Users" leftIcon={<PeopleIcon />} />
        </>
      )}
      <Menu.Item to="/guides" primaryText="Guides" leftIcon={<MenuBookIcon />} />
    </Menu>
  );
};

const AppLayout = (props: React.ComponentProps<typeof Layout>) => (
  <Layout {...props} menu={AppMenu} />
);

export default AppLayout;
