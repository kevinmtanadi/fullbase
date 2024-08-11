import { Autocomplete, AutocompleteItem, Input } from "@nextui-org/react";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { TbCirclesRelation } from "react-icons/tb";
import axiosInstance from "../../pkg/axiosInstance";
import CustomAutocomplete from "./CustomAutocomplete";

interface RelationInputProps {
  id?: string;
  name?: string;
  label: string;
  value?: string;
  onChange: (value: string) => void;
  isRequired?: boolean;
  isDisabled?: boolean;
  relatedTable: string;
}
const RelationInput = ({
  id,
  name,
  label,
  value,
  onChange,
  isRequired,
  isDisabled,
  relatedTable,
}: RelationInputProps) => {
  const { data, isLoading } = useQuery<any>({
    queryKey: ["rows", relatedTable, value],
    queryFn: async () => {
      const { data } = await axiosInstance.get(
        `/api/main/${relatedTable}/rows`,
        {
          params: {
            filter: `id LIKE '${value}%'`,
          },
        }
      );
      return data.data;
    },
  });

  return (
    <CustomAutocomplete
      id={id}
      name={name}
      value={value}
      onChange={onChange}
      options={data}
    />
    // <Autocomplete
    //   defaultSelectedKey={value}
    //   selectedKey={value}
    //   radius="sm"
    //   id={id}
    //   name={name}
    //   isDisabled={isDisabled}
    //   isRequired={isRequired}
    //   fullWidth
    //   defaultItems={[]}
    //   inputValue={searchQuery}
    //   isLoading={isLoading || !data || !data.data}
    //   items={data.data}
    //   startContent={<TbCirclesRelation />}
    //   label={label}
    //   variant="bordered"
    //   onInputChange={(value) => {
    //     setSearchQuery(value);
    //     onChange(value);
    //   }}
    // >
    //   {(item: any) => (
    //     <AutocompleteItem key={item.id}>{item.id}</AutocompleteItem>
    //   )}
    // </Autocomplete>
  );
};

export default RelationInput;
