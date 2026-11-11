import Button from 'components/atoms/Button'
import { Movie } from 'types/movie'

import styles from 'styles/MovieTableRow.module.css'

type Props = {
  movie: Movie
  isEditing: boolean
  onEdit: (movie: Movie) => void
  onDelete: (movie: Movie) => void
}

export default function MovieTableRow(props: Props) {
  const { movie, isEditing, onEdit, onDelete } = props

  const directorName =
    [movie.director?.firstname, movie.director?.lastname].filter(Boolean).join(' ') || '—'

  return (
    <tr className={isEditing ? styles.editing : undefined}>
      <td>{movie.id}</td>
      <td>{movie.title}</td>
      <td>{movie.isbn}</td>
      <td>{directorName}</td>
      <td className={styles.actions}>
        <Button onClick={() => onEdit(movie)}>Edit</Button>
        <Button variant="danger" onClick={() => onDelete(movie)}>
          Delete
        </Button>
      </td>
    </tr>
  )
}
