import {
  List, Datagrid, TextField, DateField, DeleteButton,
  Create, SimpleForm, TextInput,
  usePermissions, FunctionField,
} from 'react-admin';
import Typography from '@mui/material/Typography';

export const SshKeyList = () => {
  const { permissions } = usePermissions();
  return (
    <List sort={{ field: 'name', order: 'ASC' }}>
      <Datagrid bulkActionButtons={false}>
        <TextField source="name" />
        <FunctionField
          label="Public Key"
          render={(record: Record<string, unknown>) => {
            const key = String(record.public_key || '');
            return (
              <Typography variant="caption" sx={{ fontFamily: '"JetBrains Mono", monospace', maxWidth: 400, display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {key.length > 60 ? key.slice(0, 60) + '…' : key}
              </Typography>
            );
          }}
        />
        <DateField source="created_at" label="Created" showTime />
        {permissions === 'admin' && <DeleteButton />}
      </Datagrid>
    </List>
  );
};

export const SshKeyCreate = () => (
  <Create redirect="list">
    <SimpleForm>
      <TextInput source="name" fullWidth isRequired />
      <TextInput
        source="private_key"
        label="Private Key"
        multiline
        fullWidth
        isRequired
        minRows={6}
        sx={{ '& textarea': { fontFamily: '"JetBrains Mono", monospace', fontSize: '0.8rem' } }}
      />
      <TextInput
        source="public_key"
        label="Public Key (optional)"
        multiline
        fullWidth
        minRows={3}
        sx={{ '& textarea': { fontFamily: '"JetBrains Mono", monospace', fontSize: '0.8rem' } }}
      />
    </SimpleForm>
  </Create>
);
