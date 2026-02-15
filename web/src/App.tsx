import { Admin, Resource, CustomRoutes } from 'react-admin';
import { Route } from 'react-router-dom';

import authProvider from './authProvider';
import dataProvider from './dataProvider';
import { darkTheme, lightTheme } from './theme';
import AppLayout from './Layout';
import Dashboard from './Dashboard';

import { ProjectList, ProjectCreate, ProjectEdit, ProjectShow } from './resources/projects';
import { SshKeyList, SshKeyCreate } from './resources/sshKeys';
import { ProviderList, ProviderCreate } from './resources/providers';
import { TaskList, TaskShow } from './resources/tasks';
import { SettingsList, SettingsCreate, SettingsEdit, SettingsShow } from './resources/settings';
import { McpServerList, McpServerCreate, McpServerEdit, McpServerShow } from './resources/mcpServers';
import { UserList, UserCreate, UserEdit, UserShow } from './resources/users';
import Guides from './resources/Guides';
import KeywordsPage from './resources/keywords';

import FolderIcon from '@mui/icons-material/FolderOpen';
import KeyIcon from '@mui/icons-material/VpnKey';
import WebhookIcon from '@mui/icons-material/Webhook';
import TaskIcon from '@mui/icons-material/Assignment';
import SettingsIcon from '@mui/icons-material/Tune';
import ServerIcon from '@mui/icons-material/Dns';
import PeopleIcon from '@mui/icons-material/PeopleAlt';

const App = () => (
  <Admin
    authProvider={authProvider}
    dataProvider={dataProvider}
    dashboard={Dashboard}
    layout={AppLayout}
    darkTheme={darkTheme}
    lightTheme={lightTheme}
    defaultTheme="dark"
  >
    {(permissions) => (
      <>
        <Resource
          name="projects"
          list={ProjectList}
          create={permissions !== 'viewer' ? ProjectCreate : undefined}
          edit={permissions !== 'viewer' ? ProjectEdit : undefined}
          show={ProjectShow}
          icon={FolderIcon}
        />

        {permissions === 'admin' && (
          <Resource
            name="ssh-keys"
            list={SshKeyList}
            create={SshKeyCreate}
            icon={KeyIcon}
            options={{ label: 'SSH Keys' }}
          />
        )}

        <Resource
          name="providers"
          list={ProviderList}
          create={permissions !== 'viewer' ? ProviderCreate : undefined}
          icon={WebhookIcon}
        />

        <Resource name="keywords" />

        <Resource
          name="tasks"
          list={TaskList}
          show={TaskShow}
          icon={TaskIcon}
        />

        {permissions === 'admin' && (
          <Resource
            name="settings"
            list={SettingsList}
            create={SettingsCreate}
            edit={SettingsEdit}
            show={SettingsShow}
            icon={SettingsIcon}
          />
        )}

        {permissions === 'admin' && (
          <Resource
            name="mcp-servers"
            list={McpServerList}
            create={McpServerCreate}
            edit={McpServerEdit}
            show={McpServerShow}
            icon={ServerIcon}
            options={{ label: 'MCP Servers' }}
          />
        )}

        {permissions === 'admin' && (
          <Resource
            name="users"
            list={UserList}
            create={UserCreate}
            edit={UserEdit}
            show={UserShow}
            icon={PeopleIcon}
          />
        )}

        <CustomRoutes>
          <Route path="/guides" element={<Guides />} />
          <Route path="/keywords" element={<KeywordsPage />} />
        </CustomRoutes>
      </>
    )}
  </Admin>
);

export default App
