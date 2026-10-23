import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { fetchSurveys } from '../../api/surveys'
import { Chip } from '../../components/Chip/Chip'
import { formatDeadline } from '../../lib/format'
import styles from './Surveys.module.css'

export function Surveys() {
  const surveys = useQuery({ queryKey: ['surveys'], queryFn: fetchSurveys })

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Surveys</h1>
      {surveys.isPending && <p className={styles.notice}>Loading surveys…</p>}
      {surveys.isError && (
        <p role="alert" className={styles.error}>
          The surveys could not be loaded.
        </p>
      )}
      {surveys.isSuccess && surveys.data.length === 0 && <p className={styles.notice}>No surveys yet.</p>}
      {surveys.isSuccess && surveys.data.length > 0 && (
        <ul className={styles.list}>
          {surveys.data.map((survey) => (
            <li key={survey.id} className={`card ${styles.item}`}>
              <Link to={`/surveys/${survey.id}`} className={styles.link}>
                {survey.title}
              </Link>
              <Chip tone={survey.status === 'open' ? 'gold' : 'epic'}>
                {survey.status === 'open' ? 'Open' : 'Closed'}
              </Chip>
              <span className={styles.deadline}>
                {survey.status === 'open' ? 'Closes' : 'Closed'} {formatDeadline(survey.closesAt)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
