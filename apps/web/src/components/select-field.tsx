'use client';
import { Children, isValidElement, type ReactNode, type ComponentProps } from 'react';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from './ui/select';

// Options stay declarative; Radix supplies keyboard navigation, focus management and typeahead.
export function SelectField({
  children,
  value,
  onValueChange,
  disabled,
  ...props
}: Omit<ComponentProps<typeof SelectTrigger>, 'value' | 'onChange'> & {
  value: string;
  onValueChange: (value: string) => void;
  children: ReactNode;
}) {
  const options = Children.toArray(children).filter(
    isValidElement<{ value: string; children: ReactNode; disabled?: boolean }>,
  );
  return (
    <Select value={value} onValueChange={onValueChange} disabled={disabled || !options.length}>
      <SelectTrigger {...props} className={'app-select ' + (props.className ?? '')}>
        <SelectValue placeholder="No options available" />
      </SelectTrigger>
      <SelectContent position="popper" className="app-select-menu">
        {options.map((option) => (
          <SelectItem
            key={option.props.value}
            value={option.props.value}
            data-value={option.props.value}
            disabled={option.props.disabled}
          >
            {option.props.children}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
