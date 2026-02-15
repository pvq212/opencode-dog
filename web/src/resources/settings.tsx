import { useState, useCallback } from 'react';
import {
  List, Datagrid, TextField, DeleteButton, EditButton,
  Edit, SimpleForm, TextInput,
  Show, SimpleShowLayout,
  usePermissions, FunctionField, useRecordContext, useNotify, useRefresh,
  Create,
} from 'react-admin';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';
import Chip from '@mui/material/Chip';
import Editor from '@monaco-editor/react';

const JSON_CONFIG_KEYS = ['opencode_auth_json', 'opencode_config_json', 'opencode_ohmy_json'];

const isJsonConfig = (key: string) => JSON_CONFIG_KEYS.includes(key);

const ValueDisplay = () => {
  const record = useRecordContext();
  if (!record) return null;
  const val = String(record.value ?? '');
  const isJson = isJsonConfig(String(record.key || record.id));

  if (isJson) {
    return (
      <Chip label="JSON Config (click edit)" size="small" color="secondary" variant="outlined" />
    );
  }

  return (
    <Typography
      variant="caption"
      sx={{
        fontFamily: '"JetBrains Mono", monospace',
        maxWidth: 400,
        display: 'block',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
      }}
    >
      {val.length > 80 ? val.slice(0, 80) + '…' : val}
    </Typography>
  );
};

export const SettingsList = () => {
  const { permissions } = usePermissions();
  return (
    <List sort={{ field: 'key', order: 'ASC' }}>
      <Datagrid bulkActionButtons={false} rowClick="edit">
        <TextField source="key" label="Key" />
        <FunctionField label="Value" render={() => <ValueDisplay />} />
        {permissions === 'admin' && <EditButton />}
        {permissions === 'admin' && <DeleteButton />}
      </Datagrid>
    </List>
  );
};

export const SettingsCreate = () => (
  <Create redirect="list">
    <SimpleForm>
      <TextInput source="key" fullWidth isRequired />
      <TextInput source="value" fullWidth multiline minRows={3} />
    </SimpleForm>
  </Create>
);

const MonacoJsonEditor = ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
  <Box sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, overflow: 'hidden' }}>
    <Editor
      height="60vh"
      defaultLanguage="json"
      value={value}
      onChange={(v) => onChange(v || '')}
      theme="vs-dark"
      options={{
        minimap: { enabled: true },
        fontSize: 13,
        fontFamily: '"JetBrains Mono", monospace',
        lineNumbers: 'on',
        scrollBeyondLastLine: false,
        tabSize: 2,
        wordWrap: 'on',
        formatOnPaste: true,
      }}
    />
  </Box>
);

export const SettingsEdit = () => {
  const [jsonValue, setJsonValue] = useState<string | null>(null);

  return (
    <Edit
      mutationMode="pessimistic"
      transform={(data: Record<string, unknown>) => {
        if (jsonValue !== null) {
          return { ...data, value: jsonValue };
        }
        return data;
      }}
    >
      <SimpleForm>
        <TextInput source="key" fullWidth disabled />
        <FunctionField
          label="Value"
          render={(record: Record<string, unknown>) => {
            const key = String(record.key || record.id);
            if (isJsonConfig(key)) {
              const initial = jsonValue ?? String(record.value ?? '{}');
              return (
                <Box sx={{ width: '100%' }}>
                  <Typography variant="subtitle2" sx={{ mb: 1 }}>
                    JSON Configuration Editor
                  </Typography>
                  <MonacoJsonEditor
                    value={initial}
                    onChange={(v) => setJsonValue(v)}
                  />
                </Box>
              );
            }
            return <TextInput source="value" fullWidth multiline minRows={3} />;
          }}
        />
      </SimpleForm>
    </Edit>
  );
};

export const SettingsShow = () => (
  <Show>
    <SimpleShowLayout>
      <TextField source="key" />
      <FunctionField
        label="Value"
        render={(record: Record<string, unknown>) => {
          const key = String(record.key || record.id);
          const val = String(record.value ?? '');
          if (isJsonConfig(key)) {
            return (
              <Box sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, overflow: 'hidden' }}>
                <Editor
                  height="50vh"
                  defaultLanguage="json"
                  value={val}
                  theme="vs-dark"
                  options={{
                    readOnly: true,
                    minimap: { enabled: true },
                    fontSize: 13,
                    fontFamily: '"JetBrains Mono", monospace',
                  }}
                />
              </Box>
            );
          }
          return (
            <Typography sx={{ fontFamily: '"JetBrains Mono", monospace', whiteSpace: 'pre-wrap' }}>
              {val}
            </Typography>
          );
        }}
      />
    </SimpleShowLayout>
  </Show>
);
