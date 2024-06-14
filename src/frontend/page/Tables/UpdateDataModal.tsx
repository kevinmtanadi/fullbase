import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  Divider,
  ModalFooter,
  Button,
  BreadcrumbItem,
  Breadcrumbs,
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@nextui-org/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import TextInput from "../../components/Inputs/TextInput";
import DatetimeInput from "../../components/Inputs/DatetimeInput";
import NumberInput from "../../components/Inputs/NumberInput";
import BoolInput from "../../components/Inputs/BoolInput";
import { useFormik } from "formik";
import { toast } from "react-toastify";
import { useEffect } from "react";
import { parseAbsoluteToLocal } from "@internationalized/date";
import { BsThreeDots } from "react-icons/bs";
import { FaRegTrashAlt } from "react-icons/fa";

interface UpdateDataModalProps {
  isOpen: boolean;
  onClose: () => void;
  tableName: string;
  id: string;
}

const UpdateDataModal = ({
  isOpen,
  onClose,
  tableName,
  id,
}: UpdateDataModalProps) => {
  const { data: columns } = useQuery<any[]>({
    queryKey: ["columns", tableName],
    queryFn: async () => {
      const res = await axios.get(`/api/db/columns/${tableName}`);
      return res.data;
    },
  });

  const { data, isLoading } = useQuery<any>({
    queryKey: ["data", tableName, id],
    queryFn: async () => {
      const res = await axios.get(`/api/db/table/${tableName}/${id}`);
      return res.data;
    },
  });

  const renderInputField = (column: any, formik: any) => {
    switch (column.type) {
      case "TEXT":
        return (
          <TextInput
            isDisabled={isLoading}
            id={column.name}
            name={column.name}
            isRequired={column.notnull === 0}
            key={column.cid}
            label={column.name}
            value={formik.values ? formik.values[column.name] : ""}
            onChange={formik.handleChange}
          />
        );
      case "REAL":
      case "INTEGER":
        return (
          <NumberInput
            isDisabled={isLoading}
            id={column.name}
            name={column.name}
            isRequired={column.notnull === 0}
            key={column.cid}
            label={column.name}
            value={formik.values ? formik.values[column.name] : ""}
            onChange={formik.handleChange}
          />
        );
      case "BOOLEAN":
        return (
          <BoolInput
            isDisabled={isLoading}
            id={column.name}
            name={column.name}
            key={column.cid}
            label={column.name}
            isSelected={formik.values ? formik.values[column.name] : ""}
            onChange={formik.handleChange}
          />
        );
      case "DATETIME":
      case "TIMESTAMP":
        return (
          <DatetimeInput
            isDisabled={isLoading}
            id={column.name}
            name={column.name}
            isRequired={column.notnull === 0}
            key={column.cid}
            label={column.name}
            value={
              formik.values
                ? parseAbsoluteToLocal(new Date().toISOString())
                : undefined
            }
            onChange={formik.handleChange}
          />
        );
      default:
        return (
          <TextInput
            isDisabled={isLoading}
            id={column.name}
            name={column.name}
            isRequired={column.notnull === 0}
            key={column.cid}
            label={column.name}
            value={formik.values ? formik.values[column.name] : ""}
            onChange={formik.handleChange}
          />
        );
    }
  };

  const queryClient = useQueryClient();

  const { mutateAsync } = useMutation({
    mutationFn: async (data: any) => {
      const res = await axios.put(`/api/db/row/update`, {
        table_name: tableName,
        id: id,
        data: data,
      });
      return res.data;
    },
    onSuccess: () => {
      queryClient.refetchQueries({
        queryKey: ["rows", tableName],
      });
      onClose();
    },
  });

  const formik = useFormik({
    enableReinitialize: true,
    initialValues: data,
    onSubmit: async (values) => {
      toast.promise(mutateAsync(values), {
        pending: "Updating data...",
        success: "Data updated successfully",
        error: "Error when updating data",
      });
    },
  });

  useEffect(() => {
    if (data) {
      formik.setValues(data);
    }
  }, [data]);

  const { mutateAsync: deleteMutation } = useMutation({
    mutationFn: async () => {
      const res = await axios.delete(`/api/db/row/${tableName}/${id}`);
      return res.data;
    },
    onSuccess: () => {
      queryClient.refetchQueries({
        queryKey: ["rows", tableName],
      });
      onClose();
    },
  });

  const handleDelete = async () => {
    toast.promise(deleteMutation(), {
      pending: "Deleting data...",
      success: "Data deleted successfully",
      error: "Error when deleting data",
    });
  };

  return (
    <>
      <Modal radius="sm" size="2xl" isOpen={isOpen} onClose={onClose}>
        <ModalContent>
          <form onSubmit={formik.handleSubmit}>
            <ModalHeader className="font-normal">
              <div className="flex gap-2 items-center">
                <Breadcrumbs
                  separator="/"
                  size="lg"
                  isDisabled
                  className="text-lg px-1 font-semibold"
                >
                  <BreadcrumbItem>{tableName}</BreadcrumbItem>
                  <BreadcrumbItem>{id}</BreadcrumbItem>
                </Breadcrumbs>
                <Popover placement="bottom" radius="sm">
                  <PopoverTrigger>
                    <Button className="bg-transparent hover:bg-default-100 h-7 p-0 w-7 min-w-0">
                      <BsThreeDots />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent>
                    <Button
                      onClick={handleDelete}
                      className="bg-transparent min-w-0 w-full p-0"
                    >
                      <div className="flex gap-2 justify-between w-full items-center">
                        <p className="font-semibold text-red-500">Delete</p>
                        <FaRegTrashAlt className="text-red-500" />
                      </div>
                    </Button>
                  </PopoverContent>
                </Popover>
              </div>
            </ModalHeader>
            <ModalBody className="mb-3">
              <div className="flex flex-col gap-4">
                {columns?.map((column) => renderInputField(column, formik))}
              </div>
            </ModalBody>
            <Divider />
            <ModalFooter>
              <Button
                radius="sm"
                onClick={onClose}
                className="bg-transparent hover:bg-slate-100"
              >
                Cancel
              </Button>
              <Button
                type="submit"
                className="w-[125px] bg-slate-950 text-white font-semibold"
                radius="sm"
              >
                Save
              </Button>
            </ModalFooter>
          </form>
        </ModalContent>
      </Modal>
    </>
  );
};

export default UpdateDataModal;
