import { DatePicker, DateValue } from "@nextui-org/react";
import {
  CalendarDate,
  CalendarDateTime,
  parseDate,
  parseDateTime,
  ZonedDateTime,
} from "@internationalized/date";
import { useEffect, useState } from "react";
interface DateInputProps {
  id?: string;
  name?: string;
  label: string;
  onChange: (value: DateValue) => void;
  value?: string;
  isRequired?: boolean;
  isDisabled?: boolean;
}
const DatetimeInput = ({
  id,
  name,
  label,
  onChange,
  value,
  isRequired,
  isDisabled,
}: DateInputProps) => {
  const [date, setDate] = useState<DateValue | undefined>(
    value ? parseDateTime(value) : undefined
  );

  return (
    <DatePicker
      variant="bordered"
      radius="sm"
      id={id}
      name={name}
      showMonthAndYearPickers
      hideTimeZone
      value={date}
      onChange={(value) => {
        setDate(value);
        onChange(value);
      }}
      label={label}
      granularity="second"
      isRequired={isRequired}
      isDisabled={isDisabled}
      hourCycle={24}
    />
  );
};

export default DatetimeInput;
