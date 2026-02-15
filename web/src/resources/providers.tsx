import { useState } from 'react';
import {
  List, Datagrid, TextField, BooleanField, DeleteButton,
  Create, SimpleForm, TextInput, SelectInput,
  usePermissions, FunctionField, useRecordContext, useListContext,
  FilterForm, FilterButton, TopToolbar, CreateButton,
} from 'react-admin';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Typography from '@mui/material/Typography';
import Alert from '@mui/material/Alert';
import Editor from '@monaco-editor/react';

const providerTypeChoices = [
  { id: 'gitlab', name: 'GitLab' },
  { id: 'slack', name: 'Slack' },
  { id: 'telegram', name: 'Telegram' },
];

const providerColors: Record<string, 'warning' | 'info' | 'primary' | 'default'> = {
  gitlab: 'warning',
  slack: 'info',
  telegram: 'primary',
};

const WebhookUrlField = () => {
  const record = useRecordContext();
  if (!record?.webhook_path) return null;
  const base = window.location.origin;
  const url = `${base}${record.webhook_path}`;
  return (
    <Typography variant="caption" sx={{ fontFamily: '"JetBrains Mono", monospace', wordBreak: 'break-all' }}>
      {url}
    </Typography>
  );
};

const ProviderListActions = () => {
  const { permissions } = usePermissions();
  return (
    <TopToolbar>
      <FilterButton />
      {permissions !== 'viewer' && <CreateButton />}
    </TopToolbar>
  );
};

const ProviderFilters = [
  <TextInput key="projectId" source="projectId" label="Project ID" alwaysOn />,
  <SelectInput key="provider_type" source="provider_type" choices={providerTypeChoices} />,
];

export const ProviderList = () => {
  const { permissions } = usePermissions();
  return (
    <List filters={ProviderFilters} actions={<ProviderListActions />}>
      <Datagrid bulkActionButtons={false}>
        <FunctionField
          label="Type"
          render={(record: Record<string, unknown>) => (
            <Chip
              label={String(record.provider_type || '').toUpperCase()}
              size="small"
              color={providerColors[String(record.provider_type)] || 'default'}
              variant="filled"
            />
          )}
        />
        <TextField source="webhook_path" label="Webhook Path" />
        <WebhookUrlField />
        <BooleanField source="enabled" />
        {permissions === 'admin' && <DeleteButton />}
      </Datagrid>
    </List>
  );
};

const JsonConfigInput = ({ value, onChange }: { value: string; onChange: (val: string) => void }) => (
  <Box sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, overflow: 'hidden', height: 300 }}>
    <Editor
      height="300px"
      defaultLanguage="json"
      value={value}
      onChange={(v) => onChange(v || '{}')}
      theme="vs-dark"
      options={{
        minimap: { enabled: false },
        fontSize: 13,
        fontFamily: '"JetBrains Mono", monospace',
        lineNumbers: 'on',
        scrollBeyondLastLine: false,
        tabSize: 2,
      }}
    />
  </Box>
);

export const ProviderCreate = () => {
  const [config, setConfig] = useState('{}');

  return (
    <Create redirect="list" transform={(data: Record<string, unknown>) => ({
      ...data,
      config: JSON.parse(config),
    })}>
      <SimpleForm>
        <TextInput source="projectId" label="Project ID" fullWidth isRequired />
        <SelectInput source="provider_type" choices={providerTypeChoices} isRequired fullWidth />
        <Box sx={{ width: '100%', mb: 2 }}>
          <Typography variant="body2" sx={{ mb: 1 }}>Configuration (JSON)</Typography>
          <JsonConfigInput value={config} onChange={setConfig} />
        </Box>
        <TextInput source="webhook_secret" label="Webhook Secret" fullWidth />
        <TextInput source="webhook_path" label="Webhook Path (auto-generated if blank)" fullWidth />
        <Alert severity="info" sx={{ width: '100%' }}>
          The full webhook URL will be: <strong>{window.location.origin}/hook/{'<type>/<project_prefix>'}</strong>
        </Alert>
      </SimpleForm>
    </Create>
  );
};
