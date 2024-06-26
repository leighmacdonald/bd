import { FieldState, Updater } from '@tanstack/react-form';

export interface FieldProps<T = string> {
    disabled?: boolean;
    readonly label?: string;
    state: FieldState<T>;
    handleChange: (updater: Updater<T>) => void;
    handleBlur: () => void;
    readonly fullwidth?: boolean;
    onChange?: (value: T) => void;
    multiline?: boolean;
    rows?: number;
    placeholder?: string;
    tooltip?: string;
}
