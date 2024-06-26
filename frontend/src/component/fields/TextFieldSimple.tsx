import TextField from '@mui/material/TextField';
import { FieldProps } from './common.ts';

export const TextFieldSimple = <T,>({
    label,
    state,
    handleChange,
    handleBlur,
    fullwidth = true,
    disabled = false,
    rows = 1,
    placeholder = undefined
}: FieldProps<T>) => {
    return (
        <TextField
            multiline={rows > 1}
            rows={rows > 1 ? rows : undefined}
            disabled={disabled}
            fullWidth={fullwidth}
            label={label}
            value={state.value}
            placeholder={placeholder}
            onChange={(e) => handleChange(e.target.value as T)}
            onBlur={handleBlur}
            variant="outlined"
            error={state.meta.touchedErrors.length > 0}
            helperText={state.meta.touchedErrors}
        />
    );
};
