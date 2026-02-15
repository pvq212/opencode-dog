import {
  List, Datagrid, TextField, BooleanField, DeleteButton, EditButton, ShowButton,
  Create, Edit, SimpleForm, TextInput, SelectInput, BooleanInput,
  Show, SimpleShowLayout,
  usePermissions, useNotify, useRefresh, FunctionField, useRecordContext,
} from 'react-admin';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import DownloadIcon from '@mui/icons-material/CloudDownload';
import { installMcpServer } from '../dataProvider';

const statusColors: Record<string, 'success' | 'warning' | 'error' | 'default' | 'info'> = {
  installed: 'success',
  installing: 'warning',
  failed: 'error',
  pending: 'info',
};

const InstallButton = () => {
  const record = useRecordContext();
  const notify = useNotify();
  const refresh = useRefresh();

  if (!record) return null;

  const handleInstall = async () => {
    try {
      await installMcpServer(record.id as string);
      notify('Install triggered', { type: 'success' });
      refresh();
    } catch (e) {
      notify(`Install failed: ${e instanceof Error ? e.message : 'Unknown error'}`, { type: 'error' });
    }
  };

  return (
    <Button
      size="small"
      startIcon={<DownloadIcon />}
      onClick={handleInstall}
      variant="outlined"
      color="primary"
    >
      Install
    </Button>
  );
};

const typeChoices = [
  { id: 'npm', name: 'NPM' },
  { id: 'binary', name: 'Binary' },
];

export const McpServerList = () => {
  const { permissions } = usePermissions();
  return (
    <List sort={{ field: 'name', order: 'ASC' }}>
      <Datagrid bulkActionButtons={false}>
        <TextField source="name" />
        <TextField source="package" label="Package" />
        <FunctionField
          label="Type"
          render={(record: Record<string, unknown>) => (
            <Chip label={String(record.type || '').toUpperCase()} size="small" variant="outlined" />
          )}
        />
        <FunctionField
          label="Status"
          render={(record: Record<string, unknown>) => (
            <Chip
              label={String(record.status || 'pending')}
              size="small"
              color={statusColors[String(record.status)] || 'default'}
              variant="filled"
            />
          )}
        />
        <BooleanField source="enabled" />
        <InstallButton />
        {permissions === 'admin' && <EditButton />}
        <ShowButton />
        {permissions === 'admin' && <DeleteButton />}
      </Datagrid>
    </List>
  );
};

export const McpServerCreate = () => (
  <Create redirect="list">
    <SimpleForm>
      <TextInput source="name" fullWidth isRequired />
      <SelectInput source="type" choices={typeChoices} isRequired fullWidth />
      <TextInput source="package" label="Package" fullWidth isRequired helperText="NPM package name or binary path" />
      <TextInput source="command" fullWidth helperText="Override command (optional)" />
      <TextInput
        source="args"
        label="Arguments (JSON array)"
        fullWidth
        defaultValue="[]"
        sx={{ '& input': { fontFamily: '"JetBrains Mono", monospace' } }}
      />
      <TextInput
        source="env"
        label="Environment (JSON object)"
        fullWidth
        defaultValue="{}"
        sx={{ '& input': { fontFamily: '"JetBrains Mono", monospace' } }}
      />
    </SimpleForm>
  </Create>
);

export const McpServerEdit = () => (
  <Edit>
    <SimpleForm>
      <TextInput source="name" fullWidth isRequired />
      <SelectInput source="type" choices={typeChoices} isRequired fullWidth />
      <TextInput source="package" label="Package" fullWidth isRequired />
      <TextInput source="command" fullWidth />
      <TextInput
        source="args"
        label="Arguments (JSON array)"
        fullWidth
        sx={{ '& input': { fontFamily: '"JetBrains Mono", monospace' } }}
      />
      <TextInput
        source="env"
        label="Environment (JSON object)"
        fullWidth
        sx={{ '& input': { fontFamily: '"JetBrains Mono", monospace' } }}
      />
      <BooleanInput source="enabled" />
    </SimpleForm>
  </Edit>
);

export const McpServerShow = () => (
  <Show>
    <SimpleShowLayout>
      <TextField source="id" />
      <TextField source="name" />
      <TextField source="type" />
      <TextField source="package" label="Package" />
      <TextField source="command" />
      <FunctionField
        label="Arguments"
        render={(record: Record<string, unknown>) => (
          <Typography variant="caption" sx={{ fontFamily: '"JetBrains Mono", monospace' }}>
            {typeof record.args === 'string' ? record.args : JSON.stringify(record.args)}
          </Typography>
        )}
      />
      <FunctionField
        label="Environment"
        render={(record: Record<string, unknown>) => (
          <Typography variant="caption" sx={{ fontFamily: '"JetBrains Mono", monospace' }}>
            {typeof record.env === 'string' ? record.env : JSON.stringify(record.env)}
          </Typography>
        )}
      />
      <FunctionField
        label="Status"
        render={(record: Record<string, unknown>) => (
          <Chip
            label={String(record.status || 'pending')}
            color={statusColors[String(record.status)] || 'default'}
            variant="filled"
          />
        )}
      />
      <BooleanField source="enabled" />
      <FunctionField
        label="Error Message"
        render={(record: Record<string, unknown>) =>
          record.error_message ? (
            <Box sx={{ p: 2, borderRadius: 1, bgcolor: 'error.main', color: 'error.contrastText' }}>
              <Typography variant="body2" sx={{ fontFamily: '"JetBrains Mono", monospace' }}>
                {String(record.error_message)}
              </Typography>
            </Box>
          ) : (
            <Typography color="text.secondary">None</Typography>
          )
        }
      />
      <Box sx={{ mt: 2 }}>
        <InstallButton />
      </Box>
    </SimpleShowLayout>
  </Show>
);
